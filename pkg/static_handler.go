package webglue

import (
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	jsn "github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

const (
	WebgluePlaceholder = "{WEBGLUE}"
	DefaultIndexHtml   = `
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

type StaticHandler struct {
	indexHtml   string
	cachedFiles map[string][]byte
	devFiles    map[string]string
}

func (handler *StaticHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	webPath := request.URL.Path

	filePath, ok := handler.devFiles[webPath]
	if ok {
		http.ServeFile(writer, request, filePath)
		return
	}

	header := writer.Header()

	data, ok := handler.cachedFiles[webPath]
	if ok {
		header.Set("Content-Type", mime.TypeByExtension(filepath.Ext(webPath)))
		writer.Write(data)
		return
	}

	header.Set("Content-Type", "text/html; charset=utf-8")
	writer.Write([]byte(handler.indexHtml))
}

func newStaticHandler(allModules []*Module, indexHtml string) (*StaticHandler, error) {

	if indexHtml == "" {
		indexHtml = DefaultIndexHtml
	}

	refsCss := []string{}
	refsJs := []string{}

	cachedFiles := map[string][]byte{}
	devFiles := map[string]string{}

	mini := minify.New()
	mini.AddFunc(".css", css.Minify)
	mini.AddFunc(".html", html.Minify)
	mini.AddFunc(".svg", svg.Minify)
	mini.AddFunc(".js", js.Minify)
	mini.AddFunc(".json", jsn.Minify)
	mini.AddFunc(".xml", xml.Minify)

	anyDev := false

	for _, module := range allModules {

		if module.Resources == nil {
			continue
		}

		devPath := os.Getenv(strings.ToUpper(module.Name) + "_DEV")

		root := ""
		fs.WalkDir(module.Resources, ".", func(filePath string, entry fs.DirEntry, err error) error {

			if err != nil {
				return err
			}

			if filePath == "." {
				return nil
			}

			if root == "" {
				if !entry.IsDir() {
					return errors.New("first entry is not a directory")
				}
				root = filePath
				return nil
			}

			if !entry.IsDir() {

				webPath := filePath[len(root)+1:]

				if devPath != "" {
					devFiles["/"+webPath] = devPath + "/" + webPath
					anyDev = true
				} else {
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

				if strings.HasSuffix(webPath, ".css") {
					refsCss = append(refsCss, webPath)
				}

				if strings.HasSuffix(webPath, ".js") {
					refsJs = append(refsJs, webPath)
				}

			}

			return nil
		})

	}

	sort.Strings(refsCss)
	sort.Strings(refsJs)

	genCode := ""
	for _, cssFile := range refsCss {
		genCode += "\t\t<link rel=\"stylesheet\" href=\"" + cssFile + "\">\n"
	}

	genCode += "\t\t<script type=\"importmap\">\n\t\t\t{\n\t\t\t\t\"imports\": {\n"

	for i, jsFile := range refsJs {
		genCode += "\t\t\t\t\t\"" + jsFile[:len(jsFile)-3] + "\": \"./" + jsFile + "\""
		if i < len(refsJs)-1 {
			genCode += ","
		}
		genCode += "\n"
	}

	genCode += "\t\t\t\t}\n\t\t\t}\n\t\t</script>"

	indexHtml = strings.ReplaceAll(indexHtml, WebgluePlaceholder, genCode)

	if !anyDev {
		reader := strings.NewReader(indexHtml)
		writer := &strings.Builder{}
		err := mini.Minify(".html", writer, reader)
		if err == nil {
			indexHtml = writer.String()
		}
	}

	return &StaticHandler{
		indexHtml:   indexHtml,
		cachedFiles: cachedFiles,
		devFiles:    devFiles,
	}, nil

}
