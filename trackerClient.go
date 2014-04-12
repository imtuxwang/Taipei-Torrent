package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"strconv"
)

// Code to talk to trackers.
// Implements BEP 12 Multitracker Metadata Extension

type ClientStatusReport struct {
	Event      string
	InfoHash   string
	PeerId     string
	Port       int
	Uploaded   int64
	Downloaded int64
	Left       int64
}

func startTrackerClient(announce string, announceList [][]string, trackerInfoChan chan *TrackerResponse, reports chan ClientStatusReport) {
	if announce != "" && announceList == nil {
		// Convert the plain announce into an announceList to simplify logic
		announceList = [][]string{[]string{announce}}
	}

	if announceList != nil {
		announceList = shuffleAnnounceList(announceList)
	}

	go func() {
		for report := range reports {
			tr := queryTrackers(announceList, report)
			if tr != nil {
				trackerInfoChan <- tr
			}
		}
	}()
}

// Deep copy announcelist and shuffle each level.
func shuffleAnnounceList(announceList [][]string) (result [][]string) {
	result = make([][]string, len(announceList))
	for i, level := range announceList {
		result[i] = shuffleAnnounceListLevel(level)
	}
	return
}

func shuffleAnnounceListLevel(level []string) (shuffled []string) {
	items := len(level)
	shuffled = make([]string, items)
	perm := rand.Perm(items)
	for i, v := range perm {
		shuffled[v] = level[i]
	}
	return
}

func queryTrackers(announceList [][]string, report ClientStatusReport) (tr *TrackerResponse) {
	for _, level := range announceList {
		for i, tracker := range level {
			var err error
			tr, err = queryTracker(report, tracker)
			if err == nil {
				// Move successful tracker to front of slice for next announcement
				// cycle.
				copy(level[1:i+1], level[0:i])
				level[0] = tracker
				return
			}
		}
	}
	log.Println("Error: Did not successfully contact a tracker %v", announceList)
	return
}

func queryTracker(report ClientStatusReport, trackerUrl string) (tr *TrackerResponse, err error) {
	u, err := url.Parse(trackerUrl)
	if err != nil {
		log.Println("Error: Invalid announce URL(", trackerUrl, "):", err)
		return
	}

	uq := u.Query()
	uq.Add("info_hash", report.InfoHash)
	uq.Add("peer_id", report.PeerId)
	uq.Add("port", strconv.Itoa(report.Port))
	uq.Add("uploaded", strconv.FormatInt(report.Uploaded, 10))
	uq.Add("downloaded", strconv.FormatInt(report.Downloaded, 10))
	uq.Add("left", strconv.FormatInt(report.Left, 10))
	uq.Add("compact", "1")

	if report.Event != "" {
		uq.Add("event", report.Event)
	}

	// This might reorder the existing query string in the Announce url
	// This might break some broken trackers that don't parse URLs properly.

	u.RawQuery = uq.Encode()

	tr, err = getTrackerInfo(u.String())
	if tr == nil || err != nil {
		log.Println("Error: Could not fetch tracker info:", err)
	} else if tr.FailureReason != "" {
		log.Println("Error: Tracker returned failure reason:", tr.FailureReason)
		err = fmt.Errorf("tracker failure %s", tr.FailureReason)
	}
	return
}
