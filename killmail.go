package main

import (
	"log"
	"time"

	esi "github.com/w9jds/go.esi"
	zkb "github.com/w9jds/zkb"
)

type KillMail struct {
	*esi.KillMail
	Names map[uint32]esi.NameRef `json:"names,omitempty"`
}

func fetch() {
	log.Println("Concord Alerts have started processing!")
	queueID := getEnv("UNIQUE_QUEUE_ID", false)

	for {
		bundle, error := zkbClient.GetRedisItem(queueID)
		if error != nil || bundle.ID == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		go process(bundle)
	}
}

func process(redis *zkb.RedisResponse) {
	killMail, _, err := esiClient.GetKillMail(redis.ID, redis.Zkb.Hash, false)
	if err != nil {
		log.Println("Error pulling killmail from esi: ", err)
		return
	}

	if !isRecentKill(killMail) {
		return
	}

	ids := getUniqueIds(killMail)

	killboards := storage.FindKillboards(ids)
	watchers := storage.FindSystemWatch(uint(killMail.SystemID), ids, redis.Zkb.TotalValue)

	if len(killboards) > 0 || len(watchers) > 0 {
		names := storage.FindNames(ids)

		for _, killboard := range killboards {
			go postKillBoardMail(killboard, killMail, names, redis.Zkb.TotalValue)
		}
		for _, watcher := range watchers {
			go postSystemWatchMail(watcher, killMail, names, redis.Zkb.TotalValue)
		}
	}
}

func isRecentKill(killMail *esi.KillMail) bool {
	now := time.Now()
	occurance, err := time.Parse(time.RFC3339, killMail.Time)
	if err != nil {
		log.Println(err)
		return false
	}

	diff := now.Sub(occurance)
	minutes := diff.Minutes()
	return minutes <= 60
}

func getUniqueIds(killMail *esi.KillMail) []uint {
	unique := make(map[uint]struct{})
	unique[uint(killMail.Victim.ID)] = struct{}{}
	unique[uint(killMail.SystemID)] = struct{}{}
	unique[uint(killMail.Victim.ShipTypeID)] = struct{}{}
	unique[uint(killMail.Victim.CorporationID)] = struct{}{}

	if killMail.Victim.AllianceID != 0 {
		unique[uint(killMail.Victim.AllianceID)] = struct{}{}
	}

	for _, attacker := range killMail.Attackers {
		if _, ok := unique[uint(attacker.ID)]; !ok {
			unique[uint(attacker.ID)] = struct{}{}
		}
		if _, ok := unique[uint(attacker.ShipTypeID)]; !ok {
			unique[uint(attacker.ShipTypeID)] = struct{}{}
		}
		if _, ok := unique[uint(attacker.CorporationID)]; !ok {
			unique[uint(attacker.CorporationID)] = struct{}{}
		}

		if attacker.AllianceID != 0 {
			if _, ok := unique[uint(attacker.AllianceID)]; !ok {
				unique[uint(attacker.AllianceID)] = struct{}{}
			}
		}
	}

	ids := []uint{}
	for key := range unique {
		ids = append(ids[:], key)
	}

	return ids
}
