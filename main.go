package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/karrieretutor/b2c-tenant"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type groupIDs struct {
	Groups []string
}

type claims struct {
	ObjectID string `json:"objectId"`
}

/* Create a Prometheus Counter with the "b2c_tenant" label
   we suppose each call to this API is for one login, so we
   count it as one authentication
*/
var counterName = "b2c_authentication_count"
var counter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: counterName,
	Help: "The total count of Azure AD B2C authentications",
}, []string{"b2c_tenant", "endpoint", "path"},
)

func groupIDHandler(w http.ResponseWriter, r *http.Request) {
	t := tenant.Tenant{}
	t.ClientID = os.Getenv("B2C_CLIENT_ID")
	t.ClientSecret = os.Getenv("B2C_CLIENT_SECRET")
	t.TenantDomain = os.Getenv("B2C_TENANT_DOMAIN")

	counter.With(prometheus.Labels{"b2c_tenant": t.TenantDomain, "endpoint": r.URL.Host, "path": r.URL.Path}).Add(1)

	err := t.GetAccessToken()
	if err != nil {
		msg := "Error in setting access token: " + err.Error()
		log.Println(msg)
	}

	fmt.Println("Decoding User Object ID")

	postBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error in reading request body: %s", err.Error())
	}

	bodyString := fmt.Sprintf("%s", postBody)
	fmt.Printf("Body string: %s\n", bodyString)

	c := claims{}

	err = json.Unmarshal(postBody, &c)
	if err != nil {
		log.Printf("JSON decoding error: %s\n", err.Error())
	}
	userObjectID := c.ObjectID

	memberGroups, err := t.GetMemberGroupIDs(userObjectID)
	if err != nil {
		log.Println(err)
	}

	// If we don't get any data back (because a user doesn't exist yet, which happens directly after a fresh signup in B2C)
	// we still want to have an empty JSON array, else the B2C custom policy will fail with an Error:
	// The DataType "String" of the Value of Claim with ClaimType id "Groups" does not match the DataType "StringCollection" of ClaimType with id "aadgroups" specified in the Policy.
	if len(memberGroups) == 0 {
		memberGroups = []string{}
	}

	gids := groupIDs{Groups: memberGroups}

	log.Printf("Sending list of memberGroups back...\n")

	json.NewEncoder(w).Encode(gids)
}

func main() {

	log.Println("Starting HTTP server and listening on :8080")

	log.Println("Registering Prometheus HTTP handler on /metrics")
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Registered Prometheus counter %s", counterName)
	prometheus.MustRegister(counter)

	http.HandleFunc("/getGroupMembership/", groupIDHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
