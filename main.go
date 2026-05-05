package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/csv"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/template"
	"github.com/spf13/cobra"

	"hcshow/handlers"
	"hcshow/internal/security"
	"hcshow/middleware"
)

func main() {
	app := pocketbase.New()

	altchakey := security.GetAltchaKey()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registry := template.NewRegistry()

		// Public routes
		se.Router.GET("/login", handlers.ShowLogin(registry))
		se.Router.POST("/login", handlers.HandleLogin(app, registry))
		se.Router.GET("/register", handlers.ShowRegister(registry))
		se.Router.POST("/register", handlers.HandleRegister(app, registry, altchakey))
		se.Router.GET("/api/altcha", handlers.AltchaChallenge(altchakey))

		// Protected routes
		protected := se.Router.Group("")
		protected.BindFunc(middleware.RequireAuth(app))

		protected.GET("/", handlers.Dashboard(app, registry))
		protected.POST("/logout", handlers.Logout())

		protected.GET("/partials/exhibitors", handlers.ListExhibitors(app, registry))
		protected.GET("/partials/entries/{id}", handlers.ListEntries(app, registry))
		protected.GET("/partials/entries-form/{id}", handlers.ShowEntriesPartial(app, registry))
		protected.GET("/exhibitors/new", handlers.NewExhibitorForm(registry))
		protected.POST("/exhibitors/new", handlers.CreateExhibitor(app, registry))
		protected.GET("/exhibitors/{id}/edit", handlers.EditExhibitorForm(app, registry))
		protected.POST("/exhibitors/{id}/edit", handlers.UpdateExhibitor(app, registry))
		protected.DELETE("/exhibitors/{id}", handlers.DeleteExhibitor(app, registry))
		protected.GET("/exhibitors/{id}/entries", handlers.ShowEntries(app, registry))
		protected.POST("/exhibitors/{id}/entries", handlers.CreateEntry(app, registry))
		protected.DELETE("/exhibitors/{id}/entries/{entryid}", handlers.DeleteEntry(app, registry))
		protected.GET("/exhibitors/{id}/entries/summary_form", handlers.ShowEntrySummaryForm(app, registry))

		// Helper signup routes (add to protected group)
		protected.GET("/helper-signup", handlers.ShowHelperForm(app, registry))
		protected.POST("/helper-signup", handlers.HandleHelperSubmit(app, registry))

		// Admin routes (authenticated + admin role required)
		admin := se.Router.Group("")
		admin.BindFunc(middleware.RequireAdmin(app))

		admin.GET("/admin", handlers.AdminDashboard(app, registry))
		admin.GET("/admin/reports/entry-cards", handlers.PrintEntryCardsReport(app, registry))
		admin.GET("/admin/reports/age-group-summary", handlers.PrintExhibitorAgeGroupReport(app, registry))
		admin.GET("/admin/reports/hall-planning", handlers.PrintHallPlanningReport(app, registry))
		admin.GET("/admin/reports/pre-show-stats", handlers.PrintPreShowStatsReport(app, registry))
		admin.GET("/admin/reports/table-cards", handlers.CheckTableCards(app, registry))
		admin.GET("/admin/judge", handlers.PrintJudgeCard(registry))
		admin.GET("/admin/scanner", handlers.ShowScanner(app, registry))

		se.Router.GET("/static/{path...}", apis.Static(os.DirFS("./pb_public"), false))

		startTLSProxy(getLocalIP())

		return se.Next()
	})

	// go func() {
	// 	// Give the server a moment to start
	// 	time.Sleep(2 * time.Second)
	// 	openBrowser("http://localhost:8090/admin")
	// }()

	app.RootCmd.AddCommand(&cobra.Command{
		Use:   "import",
		Short: "Import exhibitors and entries from CSV",

		Run: func(cmd *cobra.Command, args []string) {
			runImport(app, args[0])
		},
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func generateSelfSignedCert(ip string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	pubKey := key.Public()

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"MHE Show"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IPAddresses:  []net.IP{net.ParseIP(ip), net.ParseIP("127.0.0.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, pubKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func startTLSProxy(lanIP string) {
	cert, err := generateSelfSignedCert(lanIP)
	if err != nil {
		log.Printf("TLS proxy: could not generate cert: %v", err)
		return
	}

	target, _ := url.Parse("http://localhost:8090")
	proxy := httputil.NewSingleHostReverseProxy(target)

	server := &http.Server{
		Addr: lanIP + ":8443",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
		Handler: proxy,
	}

	log.Printf("TLS proxy started at https://%s:8443", lanIP)
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Printf("TLS proxy error: %v", err)
		}
	}()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("Could not open browser: %v", err)
	}
}

func runImport(app *pocketbase.PocketBase, file string) {

	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	headers, err := reader.Read()
	if err != nil {
		panic(err)
	}

	// --- Build header index map ---
	headerIndex := map[string]int{}
	for i, h := range headers {
		key := strings.ToLower(strings.TrimSpace(h))
		headerIndex[key] = i
	}

	// --- Load categories ---
	catRecords, err := app.FindRecordsByFilter("category", "", "", 0, 0)
	if err != nil {
		panic(err)
	}

	// slug -> []category records
	categoryMap := map[string][]*core.Record{}
	catIdToSlug := map[string]string{}

	for _, c := range catRecords {
		slug := strings.ToLower(strings.TrimSpace(c.GetString("slug")))
		categoryMap[slug] = append(categoryMap[slug], c)
		catIdToSlug[c.Id] = slug
	}

	// --- Load category_age_group ---
	type CatAgeKey struct {
		Slug       string
		AgeGroupId string
	}

	catAgeMap := map[CatAgeKey]string{}

	catAgeRecords, err := app.FindRecordsByFilter("category_age_group", "", "", 0, 0)
	if err != nil {
		panic(err)
	}

	for _, r := range catAgeRecords {
		catId := r.GetString("category")
		ageGroupId := r.GetString("age_group")

		slug, ok := catIdToSlug[catId]
		if !ok {
			continue
		}

		key := CatAgeKey{
			Slug:       slug,
			AgeGroupId: ageGroupId,
		}

		catAgeMap[key] = catId
	}

	// --- Load age groups ---
	ageGroupRecords, err := app.FindRecordsByFilter("age_group", "", "min", 0, 0)
	if err != nil {
		panic(err)
	}

	type AgeGroup struct {
		Id  string
		Min int
		Max int
	}

	ageGroups := []AgeGroup{}
	for _, r := range ageGroupRecords {
		ageGroups = append(ageGroups, AgeGroup{
			Id:  r.Id,
			Min: r.GetInt("min"),
			Max: r.GetInt("max"),
		})
	}

	// --- Load show date ---
	settings, err := app.FindFirstRecordByFilter("settings", "", nil)
	if err != nil {
		panic(err)
	}

	showDate := settings.GetDateTime("show_date").Time()

	// --- Helpers ---
	ageAt := func(date time.Time, birth time.Time) int {
		age := date.Year() - birth.Year()
		if date.YearDay() < birth.YearDay() {
			age--
		}
		return age
	}

	findAgeGroup := func(age int) string {
		for _, g := range ageGroups {
			if g.Max == 0 {
				if age >= g.Min {
					return g.Id
				}
			} else if age >= g.Min && age <= g.Max {
				return g.Id
			}
		}
		return ""
	}

	// --- Load collections ---
	exhibitorCol, _ := app.FindCollectionByNameOrId("exhibitor")
	exhibitsCol, _ := app.FindCollectionByNameOrId("exhibits")

	countRows := 0
	countEntries := 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Skipping bad row:", err)
			continue
		}

		// --- Map row ---
		data := map[string]string{}
		for i, h := range headers {
			data[strings.ToLower(strings.TrimSpace(h))] = row[i]
		}

		first := strings.TrimSpace(data["childfirstname"])
		last := strings.TrimSpace(data["childlastname"])

		// --- DOB ---
		year, _ := strconv.Atoi(data["yearofbirth"])
		month, _ := strconv.Atoi(data["monthofbirth"])
		day, _ := strconv.Atoi(data["dayofbirth"])

		if year <= 1900 || year > showDate.Year() {
			fmt.Printf("Skipping invalid DOB %d/%d/%d for: %s %s\n", year, month, day, first, last)
			continue
		}

		birthTime := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		age := ageAt(showDate, birthTime)

		if age < 0 {
			fmt.Printf("Skipping future DOB for %s %s\n", first, last)
			continue
		}

		ageGroupId := findAgeGroup(age)
		if ageGroupId == "" {
			fmt.Printf("No age group for age %d (%s %s)\n", age, first, last)
		}

		// --- Extract pet extra_data ---
		petType := strings.TrimSpace(data["pettype"])
		petName := strings.TrimSpace(data["petname"])

		var extraData map[string]any
		if petType != "" || petName != "" {
			extraData = map[string]any{}
			if petType != "" {
				extraData["type"] = petType
			}
			if petName != "" {
				extraData["name"] = petName
			}
		}

		// --- Upsert exhibitor ---
		existing, _ := app.FindFirstRecordByFilter(
			"exhibitor",
			"first_name = {:first} && last_name = {:last}",
			dbx.Params{
				"first": first,
				"last":  last,
			},
		)

		var exhibitor *core.Record

		if existing != nil {
			exhibitor = existing
		} else {
			exhibitor = core.NewRecord(exhibitorCol)
			exhibitor.Set("exhibitor_id", data["entryid"])
			exhibitor.Set("first_name", first)
			exhibitor.Set("last_name", last)
			exhibitor.Set("birth_date", birthTime.Format("2006-01-02"))
			exhibitor.Set("phone", strings.ReplaceAll(data["phonenumber"], " ", ""))
			exhibitor.Set("email", strings.ReplaceAll(data["emailaddress"], " ", ""))

			if err := app.Save(exhibitor); err != nil {
				fmt.Println("Failed to save exhibitor:", err)
				continue
			}
		}

		// --- Create exhibits ---
		for slug, categories := range categoryMap {

			idx, ok := headerIndex[slug]
			if !ok {
				continue
			}

			value := strings.TrimSpace(row[idx])
			if value != "on" {
				continue
			}

			var effectiveCategoryId string

			if slug == "pet" {
				if ageGroupId == "" {
					fmt.Printf("Missing age group for pet: %s %s\n", first, last)
					continue
				}

				key := CatAgeKey{
					Slug:       slug,
					AgeGroupId: ageGroupId,
				}

				var ok bool
				effectiveCategoryId, ok = catAgeMap[key]
				if !ok {
					fmt.Printf("No category for slug=pet ageGroup=%s\n", ageGroupId)
					continue
				}
			} else {
				effectiveCategoryId = categories[0].Id
			}

			existingExhibit, _ := app.FindFirstRecordByFilter(
				"exhibits",
				"exhibitor = {:ex} && category = {:cat}",
				dbx.Params{
					"ex":  exhibitor.Id,
					"cat": effectiveCategoryId,
				},
			)

			if existingExhibit != nil {

				if existingExhibit.GetString("age_group") == "" && ageGroupId != "" {
					existingExhibit.Set("age_group", ageGroupId)
				}

				if slug == "pet" && extraData != nil {
					existingExhibit.Set("extra_data", extraData)
				}

				if err := app.Save(existingExhibit); err != nil {
					fmt.Printf("Failed to update exhibit for \"%s %s\": %v\n", first, last, err)
				}

				continue
			}

			exhibit := core.NewRecord(exhibitsCol)
			exhibit.Set("exhibitor", exhibitor.Id)
			exhibit.Set("category", effectiveCategoryId)

			if ageGroupId != "" {
				exhibit.Set("age_group", ageGroupId)
			}

			if slug == "pet" && extraData != nil {
				exhibit.Set("extra_data", extraData)
			}

			if err := app.Save(exhibit); err != nil {
				fmt.Printf("Failed to create exhibit for \"%s %s\": %v\n", first, last, err)
				continue
			}

			countEntries++
		}

		countRows++
	}

	fmt.Println("Imported rows:", countRows)
	fmt.Println("Created exhibits:", countEntries)
}
