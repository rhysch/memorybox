// chat-env creates local-only Switch/Mattermost credentials without overwriting them.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
)

func main() {
	if err := os.MkdirAll("integrations", 0755); err != nil {
		log.Fatal(err)
	}
	f, err := os.OpenFile("integrations/.env", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		log.Fatal("credentials were not changed: ", err)
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, `SWITCH_VERSION=latest
POSTGRES_HOST_PORT=55433
API_HOST_PORT=58000
GATEWAY_HOST_PORT=53000
MATTERMOST_HOST_PORT=58065
DB_USER=switch_app
DB_OWNER_USER=postgres
DB_NAME=switch
MATTERMOST_ADMIN_USER=admin
MATTERMOST_TEAM_NAME=memorybox
MATTERMOST_USER=student
GATEWAY_ADMIN_EMAIL=admin@memorybox.local
FRONTEND_BASE_URL=http://localhost:53000
TELEMETRY_ENABLED=false
THIRD_PARTY_AVATARS_ENABLED=false`)
	if err != nil {
		log.Fatal(err)
	}
	for _, name := range []string{"DB_PASSWORD", "DB_OWNER_PASSWORD", "AGENT_REGISTRATION_TOKEN", "SECRET_KEYS", "GATEWAY_ADMIN_PASSWORD", "MATTERMOST_ADMIN_PASSWORD", "MATTERMOST_USER_PASSWORD"} {
		secret := make([]byte, 32)
		if _, err = rand.Read(secret); err != nil {
			log.Fatal(err)
		}
		value := hex.EncodeToString(secret)
		if name == "SECRET_KEYS" {
			value = "local:" + value
		}
		if _, err = fmt.Fprintf(f, "%s=%s\n", name, value); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("Created integrations/.env with private local credentials.")
}
