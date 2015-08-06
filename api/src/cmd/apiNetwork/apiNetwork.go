package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "log"
    "os"
    "strings"

    "github.com/julienschmidt/httprouter"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

var (
    dbUrl string
    dbName string
    dbSession *mgo.Session
    )

func main() {
    // Set up persistence. TODO: Move to a repository implementation.
    dbUrl = getEnv("MGO_CONN", "mongodb://"mongoDev.local"")
    fmt.Println(dbUrl)
    dbName = getEnv("MGO_DB", "Networks")
    fmt.Println(dbName)

    // Set routes.
    router := httprouter.New()
    router.GET("/networks/:id", getNetworkById)
    router.POST("/networks", postNetworks)
    router.POST("/networks/search", searchNetworks)

    // Start http server. TODO: Read the port from environment.
    port := ":8081"
    fmt.Printf("Starting http server on %s\n", port)
    err := http.ListenAndServe(port, router)
    if (err != nil) {
        log.Panic(err)
    }
}

func getNetworkById(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    result := Network{}
    id := ps.ByName("id")
    if (!isEmptyOrWhitespace(id)) {
        session := cloneSession()
        defer session.Close()
        c := session.DB(dbName).C("Networks")
        err := c.FindId(bson.ObjectIdHex(id)).One(&result)
        if (err != nil) {
            err = fmt.Errorf("getNetworkById %s: %v", id, err)
            log.Print(err)
        }
    }
    outInfo, _ := json.Marshal(result)
    fmt.Fprint(w, string(outInfo))
}

func postNetworks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    d := json.NewDecoder(r.Body)
    var rb Network
    err := d.Decode(&rb)
    if (err != nil) {
        log.Print(err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    rb.Id = bson.NewObjectId()
    fmt.Println(rb)

    session := cloneSession()
    defer session.Close()
    c := session.DB(dbName).C("Networks")
    err = c.Insert(rb)
    if (err != nil) {
        log.Print(err)
        http.Error(w, err.Error(), http.StatusNotAcceptable)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

func searchNetworks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    d := json.NewDecoder(r.Body)
    var rb NetworkRequest
    err := d.Decode(&rb)
    if (err != nil) {
        log.Print(err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    fmt.Println(rb)

    op1 := bson.M{ "gatewayMacs": rb.Gateway }
    op2 := bson.M{ "devices.deviceKey": bson.M{ "$in": rb.Devices} }
    op3 := bson.M{ "$or": []interface{}{ op1, op2, } }
    op := bson.M{ "$match": op3 }

    result := []Network{}
    session := cloneSession()
    defer session.Close()
    c := session.DB(dbName).C("Networks")
    pipe := c.Pipe([]bson.M{op})
    err = pipe.All(&result)
    if (err != nil) {
        log.Print(err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    outInfo, _ := json.Marshal(result)
    fmt.Fprint(w, string(outInfo))
}

func isEmptyOrWhitespace(s string) bool {
    return (len(s) == 0 || len(strings.TrimSpace(s)) == 0)
}

func getEnv(name string, ifEmpty string) string {
    s := os.Getenv(name)
    if (len(s) == 0 || len(strings.TrimSpace(s)) == 0) {
        return ifEmpty
    }
    return s
}

func cloneSession() *mgo.Session {
    if (dbSession == nil) {
        var err error
        dbSession, err = mgo.Dial(dbUrl)
        if (err != nil) {
            log.Panic(err)
        }
    }
    return dbSession.Clone()
}

type NetworkRequest struct {
    Gateway string `bson:"gatewayMac,omitempty" json:"gatewayMac,omitempty"`
    Devices []string `bson:"deviceMacs,omitempty" json:"deviceMacs,omitempty"`
}

type Network struct {
    Id bson.ObjectId `bson:"_id,omitempty" json:"id,omitempty"`
    Name string `bson:"networkName,omitempty" json:"networkName,omitempty"`
    Gateways []string `bson:"gatewayMacs,omitempty" json:"gatewayMacs,omitempty"`
    AssociatedDeviceIds []string `bson:"associatedDeviceIds,omitempty" json:"associatedDeviceIds,omitempty"`
    Devices []Device `bson:"devices,omitempty" json:"devices,omitempty"`
}

type Device struct {
    DeviceKey string `bson:"deviceKey,omitempty" json:"deviceKey,omitempty"`
    Name string `bson:"deviceName,omitempty" json:"deviceName,omitempty"`
    MacAddress string `bson:"macAddress,omitempty" json:"macAddress,omitempty"`
    LastSeen string `bson:"LastSeen,omitempty" json:"LastSeen,omitempty"`
    Modified string `bson:"modified,omitempty" json:"modified,omitempty"`
    IsSymbi bool `bson:"isSymbiEnabled,omitempty" json:"isSymbi,omitempty"`
    IsMonitored bool `bson:"isMonitored,omitempty" json:"isMonitored,omitempty"`
}
