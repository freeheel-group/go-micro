package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"

    "github.com/julienschmidt/httprouter"

	"github.com/oschwald/maxminddb-golang"
)

var ( 
	dbGeo, dbIsp *maxminddb.Reader
	)

func main() {
	// Set up MaxMind databases.
	var err error
	dbGeo, err = openDb("data/maxMind/GeoIP2-City.mmdb")
	if (err != nil || dbGeo == nil) {
  		log.Panic(err)
	}
	defer dbGeo.Close()
	dbIsp, err = openDb("data/maxMind/GeoIP2-ISP.mmdb")
	if (err != nil || dbIsp == nil) {
  		log.Panic(err)
	}
	defer dbIsp.Close()

    // Set routes.
    router := httprouter.New()
    router.GET("/:ipAddress", getNetLocation)

	// Start http server. TODO: Read the port from environment.
    port := ":8082"
    fmt.Printf("Starting http server on %s\n", port)
    err = http.ListenAndServe(port, router)
	if err != nil {
 		log.Panic(err)
	}
}

func getNetLocation(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Get specified IP address for geo-location lookup. If no IP address
	// specified, use the IP address of the requesting device.
	var ip net.IP
	var err error
    ipAddress := ps.ByName("ipAddress")
	if (isEmptyOrWhitespace(ipAddress)) {
		ipAddress = getRequestorIp(r)
		if (err != nil) {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}	
	}

 	ip = net.ParseIP(ipAddress)
	if (ip == nil) {
		err := fmt.Errorf("Invalid IP address %s", ipAddress)
		log.Print(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	netLocation := getInfo(ip)
	outInfo, _ := json.Marshal(netLocation)
	fmt.Fprint(w, string(outInfo))
}

func isEmptyOrWhitespace(s string) bool {
	return (len(s) == 0 || len(strings.TrimSpace(s)) == 0)
}

func getRequestorIp(r *http.Request) string {
	ipProxy := r.Header.Get("x-forwarded-for")
	if (isEmptyOrWhitespace(ipProxy)) {
		ipAddress, _, _ := net.SplitHostPort(r.RemoteAddr)
		return ipAddress
	}
	ips := strings.Split(ipProxy, ", ")
	if (len(ips) > 0) {
		return ips[0]
	}
	return "Undetermined"
}

func openDb(dbPath string) (*maxminddb.Reader, error) {
	path, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	db, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return db, err
}

func getInfo(ip net.IP) (*NetLocation) {
	// TODO: Experiment with running getGeo and getIsp concurrently.
	//fmt.Println("Looking up geo-data and ISP for IP address:", ip)
	netLocation := &NetLocation{
        IpAddress: ip.String()}

		// Get geo-location data.
	geoData, err := getGeo(ip)
	if (err != nil) {
		log.Print(err)

	} else {
		const language string = "en"
		netLocation.City = geoData.City.Names[language]
		netLocation.CountryCode = geoData.Country.IsoCode
		netLocation.Country = geoData.Country.Names[language]
		subdivisions := geoData.Subdivisions
		if (len(subdivisions) > 0) {
			subdivision := subdivisions[0]
			netLocation.RegionCode = subdivision.IsoCode
			netLocation.Region = subdivision.Names[language]
		}
		netLocation.Lat = geoData.Location.Latitude
		netLocation.Lon = geoData.Location.Longitude
		netLocation.Timezone = geoData.Location.Timezone
		netLocation.Postal = geoData.Postal.Code
	}
	
	// Get ISP data.
		ispData, err := getIsp(ip)
		if (err != nil) {
		log.Print(err)
	} else {
		netLocation.Isp = ispData.Name
    }
    return netLocation
}

func getGeo(ip net.IP) (*geoData, error) {
	var record geoData
	err := dbGeo.Lookup(ip, &record)
	if err != nil {
		return nil, err
	}
	return &record, err
}

func getIsp(ip net.IP) (*ispData, error) {
	var record ispData
	err := dbIsp.Lookup(ip, &record)
	if err != nil {
		return nil, err
	}
	return &record, err
}

func getMaxmind(ip net.IP, db *maxminddb.Reader, record interface{}) error {
	err := db.Lookup(ip, &record)
	return err
}

type geoData struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Location struct {
		Latitude float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		Timezone string `maxminddb:"time_zone"`
	} `maxminddb:"location"`
    Postal struct {
		Code string `maxminddb:"code"`
    } `maxminddb:"postal"`
	Subdivisions []struct {
		IsoCode string `maxminddb:"iso_code"`
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

type ispData struct {
	Name string `maxminddb:"isp"`
}

type NetLocation struct {
    IpAddress string `json:"ipAddress,omitempty"`
    CountryCode string `json:"countryCode,omitempty"`
    Country string `json:"country,omitempty"`
    RegionCode string `json:"regionCode,omitempty"`
    Region string `json:"region,omitempty"`
    City string `json:"city,omitempty"`
    Postal string `json:"postal,omitempty"`
    Lat float64 `json:"lat,omitempty"`
    Lon float64 `json:"lon,omitempty"`
    Timezone string `json:"timezone,omitempty"`
    Isp string `json:"isp,omitempty"`
}
