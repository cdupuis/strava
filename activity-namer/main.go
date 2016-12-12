package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cdupuis/strava/activity-namer/Godeps/_workspace/src/github.com/strava/go.strava"
	"github.com/cdupuis/strava/activity-namer/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/cdupuis/strava/activity-namer/Godeps/_workspace/src/googlemaps.github.io/maps"
	"github.com/cdupuis/strava/activity-namer/persistence"
)

func main() {

	// set up command line options
	var stravaAccessToken, mapsAccessToken string
	var resetCommuteCounter bool
	flag.BoolVar(&resetCommuteCounter, "reset-commute-counter", true, "Reset Commute Counter")
	flag.StringVar(&stravaAccessToken, "strava-token", os.Getenv("STRAVA_TOKEN"), "Strava Access Token")
	flag.StringVar(&mapsAccessToken, "maps-token", os.Getenv("MAPS_TOKEN"), "Google Maps Access Token")
	flag.Parse()

	if stravaAccessToken == "" {
		fmt.Println("\nPlease provide a strava-token")

		flag.PrintDefaults()
		os.Exit(1)
	}

	if mapsAccessToken == "" {
		fmt.Println("\nPlease provide a maps-token")

		flag.PrintDefaults()
		os.Exit(1)
	}

	// open the persistence backend
	db := &persistence.DB{Store: persistence.Open()}
	defer db.Close()

	mapsClient, err := maps.NewClient(maps.WithAPIKey(mapsAccessToken))
	if err != nil {
		fmt.Printf("fatal error: %s", err)
		os.Exit(1)
	}

	client := strava.NewClient(stravaAccessToken)
	service := strava.NewCurrentAthleteService(client)
	activities, err := service.ListActivities().Do()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// reset in the internal counter if asked
	reset(db, resetCommuteCounter, activities)

	for _, e := range activities {

		fmt.Print(e.Name)

		if strings.Count(e.Name, "-") == 0 {

			city, country := geocode(mapsClient, e.StartLocation[0], e.StartLocation[1])

			name := e.StartDateLocal.Format("02/01/2006") + " " + city + ", " + country + " - "

			if e.Commute {
				name += "Commute #" + db.Increment() + " - "
			}

			name += e.Name

			s := strava.NewActivitiesService(client)
			_, err = s.Update(e.Id).Name(name).Private(e.Private).Do()
			if err != nil {
				fmt.Println(err)
			}

			fmt.Printf("... updated with new name (%s)\n", name)

		} else {
			fmt.Println("... ignored")
		}
	}
}

func reset(db *persistence.DB, resetCommuteCounter bool, activities []*strava.ActivitySummary) {
	if resetCommuteCounter {

		re, err := regexp.Compile(`.#(\d+).`)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		var commuteCounter int64

		for _, e := range activities {
			res := re.FindAllStringSubmatch(e.Name, -1)
			if res != nil {
				intCounter, err := strconv.ParseInt(res[0][1], 10, 8)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				if intCounter > commuteCounter {
					commuteCounter = intCounter
				}
			}
		}

		db.Reset(strconv.FormatInt(commuteCounter, 10))
		fmt.Printf("Resetting Commute Counter to %s\n", db.Read())
	}
}

func geocode(mapsClient *maps.Client, lat float64, lng float64) (string, string) {
	r := &maps.GeocodingRequest{
		LatLng: &maps.LatLng{Lat: lat, Lng: lng},
	}

	resp, err := mapsClient.Geocode(context.Background(), r)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var city, country string
	for _, b := range resp[0].AddressComponents {
		if stringInSlice("country", b.Types) {
			country = b.LongName
		}
		if stringInSlice("locality", b.Types) {
			city = b.LongName
		}
	}

	return city, country
}
