// Command agri-alert is the STARTING POINT for the port. It compiles and boots
// a bare Via v0.7 app with a single placeholder page. Your job (see
// ../CONTRACT.md) is to grow it into the full server-rendered port: login +
// HMAC sessions, contacts (list/create/edit/delete + validation), stations
// (list + status-change with a status log), the /map grids (sectors,
// sporulation, thermal with a day selector, battery) as GeoJSON + legends, the
// Datastar SSE actions, and the SMS broadcast.
//
// Config comes from env (all read here or wherever you load the seed):
//   PORT                 listen port (default 8080)
//   AGRI_SEED_DIR        seed directory (default ./seed): seed.sql,
//                        sector_grid.csv, monitored_sectors.json, feeds/*.json
//   AGRI_SESSION_SECRET  HMAC secret for session tokens
//   AGRI_MANAGER_PASSWORD, AGRI_VIEWER_PASSWORD, AGRI_SECURE_COOKIE
//   AGRI_SMS_URL         upstream the /sms broadcast forwards to
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-via/via"
	"github.com/go-via/via/h"
)

// Index is a placeholder composition. Replace it with the real Agri-Alert
// surface per ../CONTRACT.md.
type Index struct{}

func (p *Index) View(ctx *via.CtxR) h.H {
	return h.Div(h.Text("agri-alert: not yet ported — implement ../CONTRACT.md"))
}

func main() {
	seedDir := os.Getenv("AGRI_SEED_DIR")
	if seedDir == "" {
		seedDir = "./seed"
	}
	// TODO(port): load seedDir/seed.sql, sector_grid.csv, monitored_sectors.json,
	// and feeds/{sporulation,thermal,battery}.json into your stores.
	_ = seedDir

	app := via.New()
	via.Mount[Index](app, "/")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("agri-alert listening on :%s (seed=%s)", port, seedDir)
	log.Fatal(http.ListenAndServe(":"+port, app))
}
