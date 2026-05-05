package webglue

import (
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	json "github.com/json-iterator/go"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	jsn "github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

const (
	// WebgluePlaceholder is replaced in index.html with generated stylesheet links and import map.
	WebgluePlaceholder = "{WEBGLUE}"

	// DefaultIndexHtml is the default HTML template used when no custom template is provided.
	// It includes the webglue client library and jQuery.
	DefaultIndexHtml = `
<!DOCTYPE html>
<html>
	<head>
		<title>Loading...</title>
		<link rel="shortcut icon" href="/favicon.png?v1">
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
{WEBGLUE}
		<script type="module">
			import {start} from "webglue";
			$(document).ready(start);
		</script>
	</head>
	<body>
	</body>
</html>
	`
)

// StaticHandler serves static files (HTML, CSS, JS, images) and provides SPA fallback.
// It supports two modes:
//   - Production: serves minified, cached resources from embedded filesystems
//   - Development: serves files directly from the filesystem for hot reload
type StaticHandler struct {
	indexHtml     string            // Processed HTML with import map injected
	cachedFiles   map[string][]byte // Minified resources cached in memory
	devFiles      map[string]string // File paths for development mode
	workspacePath string            // Optional path for DevTools workspace integration
}

// DevTools integration: serve workspace configuration for Chrome DevTools when requested
type WorkspaceJson struct {
	Workspace WorkspaceJsonWorkspace `json:"workspace"`
}
type WorkspaceJsonWorkspace struct {
	Root string `json:"root"`
	UUID string `json:"uuid"`
}

// ServeHTTP handles static file requests.
// Behavior:
//   - If in dev mode and file exists -> serve from filesystem
//   - If file is cached -> serve from memory
//   - Otherwise -> serve index.html (SPA fallback)
func (handler *StaticHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	webPath := request.URL.Path

	// Development mode: serve from filesystem
	filePath, ok := handler.devFiles[webPath]
	if ok {
		http.ServeFile(writer, request, filePath)
		return
	}

	if webPath == "/.well-known/appspecific/com.chrome.devtools.json" && handler.workspacePath != "" {
		workspace := WorkspaceJson{
			Workspace: WorkspaceJsonWorkspace{
				Root: handler.workspacePath,
				UUID: stringToUUID(handler.workspacePath),
			},
		}
		writer.Header().Set(ContentTypeHeader, ContentTypeJson)
		err := json.NewEncoder(writer).Encode(workspace)
		if err != nil {
			http.Error(writer, "Failed to encode workspace JSON", http.StatusInternalServerError)
			return
		}
		return
	}

	header := writer.Header()

	// Production mode: serve from cached memory
	data, ok := handler.cachedFiles[webPath]
	if ok {
		header.Set("Content-Type", mime.TypeByExtension(filepath.Ext(webPath)))
		writer.Write(data)
		return
	}

	// SPA fallback: serve index.html for unknown paths
	header.Set("Content-Type", "text/html; charset=utf-8")
	writer.Write([]byte(handler.indexHtml))
}

// newStaticHandler creates a static file handler with automatic resource processing.
// It performs the following:
//  1. Scans all module resources
//  2. Minifies CSS, JS, HTML, JSON, SVG, XML
//  3. Generates import map for ES modules
//  4. Injects stylesheet links and import map into HTML
//  5. Caches everything in memory (unless in dev mode)
//
// Development mode is activated by setting environment variable: {MODULENAME}_DEV=/path/to/files
func newStaticHandler(allModules []*Module, indexHtml string) (*StaticHandler, error) {

	if indexHtml == "" {
		indexHtml = DefaultIndexHtml
	}

	refsCss := []string{}
	refsJs := []string{}

	cachedFiles := map[string][]byte{}
	devFiles := map[string]string{}
	anyDev := false
	workspacePath := ""

	// Set up minifier for all supported file types
	mini := minify.New()
	mini.AddFunc(".css", css.Minify)
	mini.AddFunc(".html", html.Minify)
	mini.AddFunc(".svg", svg.Minify)
	mini.AddFunc(".js", js.Minify)
	mini.AddFunc(".json", jsn.Minify)
	mini.AddFunc(".xml", xml.Minify)

	// Process resources from all modules
	for _, module := range allModules {

		if module.Resources == nil {
			continue
		}

		// Check for development mode environment variable
		// e.g., MYMODULE_DEV=/path/to/dev/files
		devPath := os.Getenv(strings.ToUpper(module.Name) + "_DEV")

		root := ""
		fs.WalkDir(module.Resources, ".", func(filePath string, entry fs.DirEntry, err error) error {

			if err != nil {
				return err
			}

			if filePath == "." {
				return nil
			}

			// First entry should be the root directory (e.g., "client")
			if root == "" {
				if !entry.IsDir() {
					return errors.New("first entry is not a directory")
				}
				root = filePath
				return nil
			}

			if !entry.IsDir() {

				// Remove root directory from path (e.g., "client/app.js" -> "app.js")
				webPath := filePath[len(root)+1:]

				if devPath != "" {
					// Development mode: map URL to filesystem path
					devFiles["/"+webPath] = devPath + "/" + webPath
					anyDev = true
					workspacePath = devPath
				} else {
					// Production mode: read, minify, and cache
					content, err := module.Resources.ReadFile(filePath)
					if err != nil {
						return err
					}

					ext := filepath.Ext(filePath)
					reader := strings.NewReader(string(content))
					writer := &strings.Builder{}

					err = mini.Minify(ext, writer, reader)
					if err == nil {
						content = []byte(writer.String())
					}

					cachedFiles["/"+webPath] = content
				}

				// Track CSS files for stylesheet link generation
				if strings.HasSuffix(webPath, ".css") {
					refsCss = append(refsCss, webPath)
				}

				// Track JS files for import map generation
				if strings.HasSuffix(webPath, ".js") {
					refsJs = append(refsJs, webPath)
				}

			}

			return nil
		})

	}

	// Sort for deterministic output
	sort.Strings(refsCss)
	sort.Strings(refsJs)

	// Generate stylesheet links
	genCode := ""
	for _, cssFile := range refsCss {
		genCode += "\t\t<link rel=\"stylesheet\" href=\"" + cssFile + "\">\n"
	}

	// Generate import map for ES modules
	// This allows: import {api} from "webglue" instead of "./webglue.js"
	genCode += "\t\t<script type=\"importmap\">\n\t\t\t{\n\t\t\t\t\"imports\": {\n"

	for i, jsFile := range refsJs {
		// Map "webglue" to "./webglue.js"
		genCode += "\t\t\t\t\t\"" + jsFile[:len(jsFile)-3] + "\": \"./" + jsFile + "\""
		if i < len(refsJs)-1 {
			genCode += ","
		}
		genCode += "\n"
	}

	genCode += "\t\t\t\t}\n\t\t\t}\n\t\t</script>"

	// Inject generated code into HTML
	indexHtml = strings.ReplaceAll(indexHtml, WebgluePlaceholder, genCode)

	// Minify HTML in production mode
	if !anyDev {
		reader := strings.NewReader(indexHtml)
		writer := &strings.Builder{}
		err := mini.Minify(".html", writer, reader)
		if err == nil {
			indexHtml = writer.String()
		}
	}

	return &StaticHandler{
		indexHtml:     indexHtml,
		cachedFiles:   cachedFiles,
		devFiles:      devFiles,
		workspacePath: workspacePath,
	}, nil

}

func stringToUUID(s string) string {
	// Simple hash function to generate a UUID-like string from the input
	hash := 0
	for _, char := range s {
		hash = int(char) + ((hash << 5) - hash)
	}
	uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		hash&0xffffffff,
		(hash>>32)&0xffff,
		((hash>>48)&0x0fff)|0x4000, // Version 4
		((hash>>64)&0x3fff)|0x8000, // Variant 1
		hash&0xffffffffffff,
	)
	return uuid
}
