package main

import (
	"database/sql"
	"fmt"
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

	return storage
}

func (storage Storage) CreateKillBoard(channelID string, guildID string, groupId uint) {
	killboard := `INSERT INTO channels (channel_id, guild_id, type, group_id) VALUES ($1, $2, 'KillBoard', $3)`

	_, err := storage.database.Exec(killboard, channelID, guildID, groupId)
	if err != nil {
		log.Println(err)
	}
}

func (storage Storage) FindKillboard(channelID string, guildID string) (uint, error) {
	var groupID uint
	query := `SELECT group_ID FROM channels WHERE type = 'KillBoard' AND channel_id=$1 AND guild_id=$2`

	row := storage.database.QueryRow(query, channelID, guildID)
	switch error := row.Scan(&groupID); error {
	case sql.ErrNoRows:
		return groupID, sql.ErrNoRows
	case nil:
		return groupID, nil
	default:
		return groupID, error
	}
}

func (storage Storage) FindKillboards(ids []uint) []KillBoard {
	killboards := []KillBoard{}
	conv := []string{}

	for _, id := range ids {
		conv = append(conv, fmt.Sprintf("%d", id))
	}

	values := strings.Join(conv[:], ",")
	query := fmt.Sprintf(`SELECT channel_id, guild_id, group_id FROM channels WHERE type = 'KillBoard' AND group_id IN (%s)`, values)

	rows, err := storage.database.Query(query)
	if err != nil {
		log.Println(err)
	} else {
		for rows.Next() {
			board, err := readKillBoard(rows)
			if err != nil {
				log.Println(err)
			} else {
				killboards = append(killboards, board)
			}
		}
	}

	return killboards
}

func (storage Storage) CreateNames(names map[uint]esi.NameRef) {
	if len(names) > 0 {
		values := "('%d','%s','%s')"

		items := []string{}
		for _, ref := range names {
			if ref.Category != "character" {
				name := strings.ReplaceAll(ref.Name, "'", "''")
				items = append(items, fmt.Sprintf(values, ref.ID, ref.Category, name))
			}
		}

		if len(items) > 0 {
			query := fmt.Sprintf(`INSERT INTO names (id, category, name) VALUES %s`, strings.Join(items, ","))

			_, err := storage.database.Exec(query)
			if err != nil {
				log.Println(query)
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
