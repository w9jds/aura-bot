package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	_ "github.com/lib/pq"
	esi "github.com/w9jds/go.esi"
)

type Storage struct {
	database *sql.DB
}

func CreateStorage() *Storage {
	password := getEnv("POSTGRES_PASSWORD", true)
	host := getEnv("POSTGRES_HOSTNAME", true)
	dbname := getEnv("POSTGRES_DB", true)
	user := getEnv("POSTGRES_USER", true)

	dsn := fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable", host, dbname, user, password)

	postgres, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	storage := &Storage{
		database: postgres,
	}

	storage.init()
	return storage
}

func (storage Storage) init() {
	contents, err := ioutil.ReadFile("./setup.sql")
	if err != nil {
		log.Panic(err)
	}

	_, err = storage.database.Exec(string(contents))
	if err != nil {
		log.Panic(err)
	}
}

func (storage Storage) CreateNames(names map[uint]esi.NameRef) {
	if len(names) > 0 {
		values := "('%d','%s','%s')"

		items := []string{}
		for _, ref := range names {
			if ref.Category != "character" {
				items = append(items, fmt.Sprintf(values, ref.ID, ref.Category, ref.Name))
			}
		}

		if len(items) > 0 {
			query := fmt.Sprintf(`INSERT INTO names (id, category, name) VALUES %s`, strings.Join(items, ","))

			_, err := storage.database.Exec(query)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (storage Storage) FindNames(ids []uint) map[uint]esi.NameRef {
	names := []esi.NameRef{}
	conv := []string{}

	for _, id := range ids {
		conv = append(conv, fmt.Sprintf("%d", id))
	}

	query := `SELECT id, category, name FROM names WHERE id IN (%s)`
	values := strings.Join(conv[:], ", ")

	rows, err := storage.database.Query(fmt.Sprintf(query, values))
	if err != nil {
		log.Println(err)
	} else {
		for rows.Next() {
			name := esi.NameRef{}

			err := rows.Scan(
				&name.ID,
				&name.Category,
				&name.Name,
			)

			if err != nil {
				log.Println(err)
			} else {
				names = append(names, name)
			}
		}
	}

	cached := map[uint]esi.NameRef{}
	for _, name := range names {
		cached[name.ID] = name
	}

	missing := []uint{}
	for _, id := range ids {
		if _, ok := cached[id]; !ok {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		resolved, err := esiClient.GetNames(missing)
		if err != nil {
			log.Println(err)
		}

		go storage.CreateNames(resolved)
		for key, value := range resolved {
			cached[key] = value
		}
	}

	return cached
}

func (storage Storage) RemoveChannel(channelID string, guildID string) {
	killboard := `DELETE FROM channels WHERE guild_id=$1 AND channel_id=$2`

	_, err := storage.database.Exec(killboard, guildID, channelID)
	if err != nil {
		log.Println(err)
	}
}

func (storage Storage) CleanupGuild(guildID string) {
	query := fmt.Sprintf(`DELETE FROM channels WHERE guild_id='%s'`, guildID)

	_, err := storage.database.Exec(query)
	if err != nil {
		log.Println(err)
	}
}
