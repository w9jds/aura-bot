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

type ChannelConfig struct {
	FilterID  uint   `json:"filter_id"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
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

func (storage Storage) CreateSystemWatch(channelID string, guildID string, systemID uint, ignoreList string, minValue float64) {
	watch := fmt.Sprintf(
		`INSERT INTO watchers (channel_id, guild_id, filter_ids, ignore_list, min_value) VALUES ($1, $2, ARRAY[%d]::int[], ARRAY[%s]::int[], $3)`,
		systemID,
		ignoreList,
	)

	_, err := storage.database.Exec(watch, channelID, guildID, minValue)
	if err != nil {
		log.Println(err)
	}
}

func (storage Storage) FindSystemWatch(systemId uint, ids []uint, value float64) []string {
	channels := []string{}
	conv := []string{}

	for _, id := range ids {
		conv = append(conv, fmt.Sprintf("%d", id))
	}

	query := fmt.Sprintf(
		`SELECT DISTINCT(channel_id) FROM watchers WHERE filter_ids && '{%d}' AND min_value < %f AND NOT ignore_list && '{%s}'`,
		systemId,
		value,
		strings.Join(conv[:], ","),
	)

	rows, err := storage.database.Query(query)
	if err != nil {
		log.Println(err)
	} else {
		for rows.Next() {
			var channel string
			err := rows.Scan(&channel)
			if err != nil {
				log.Println(err)
			} else {
				channels = append(channels, channel)
			}
		}
	}

	return channels
}

func (storage Storage) CreateKillBoard(channelID string, guildID string, groupId uint) {
	killboard := `INSERT INTO channels (channel_id, guild_id, type, filter_id) VALUES ($1, $2, 'killboard', $3)`

	_, err := storage.database.Exec(killboard, channelID, guildID, groupId)
	if err != nil {
		log.Println(err)
	}
}

func (storage Storage) FindKillboard(channelID string, guildID string) (uint, error) {
	var groupID uint
	query := `SELECT filter_id FROM channels WHERE type = 'killboard' AND channel_id=$1 AND guild_id=$2`

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

func (storage Storage) FindKillboards(ids []uint) []ChannelConfig {
	killboards := []ChannelConfig{}
	conv := []string{}

	for _, id := range ids {
		conv = append(conv, fmt.Sprintf("%d", id))
	}

	values := strings.Join(conv[:], ",")
	query := fmt.Sprintf(`SELECT channel_id, guild_id, filter_id FROM channels WHERE type = 'killboard' AND filter_id IN (%s)`, values)

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
