package webglue

import (
	"embed"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	PlaceholderCss   = "{WEBGLUE_CSS}"
	PlaceholderJs    = "{WEBGLUE_JS}"
	DefaultIndexHtml = `
<!DOCTYPE html>
<html>
	<head>
		<title>Loading...</title>
		<link rel="shortcut icon" href="/favicon.png?v1">
		<meta charset="UTF-8">
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
{WEBGLUE_CSS}{WEBGLUE_JS}		<script>
		$(document).ready(startWebglue);
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

//go:embed client
var clientResources embed.FS

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

func newStaticHandler(options Options) (*StaticHandler, error) {
	allModules := append([]Module{
		{
			Name:      "webglue",
			Resources: clientResources,
		},
	}, options.Modules...)

	indexHtml := DefaultIndexHtml
	if options.IndexHtml != "" {
		indexHtml = options.IndexHtml
	}

	refsCss := []string{}
	refsJs := []string{}

	cachedFiles := map[string][]byte{}
	devFiles := map[string]string{}

	for _, module := range allModules {

		println("module", module.Name)

		devPath := os.Getenv(strings.ToUpper(module.Name) + "_DEV")

		println("devPath", devPath)

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
				} else {
					cachedFiles["/"+webPath], err = module.Resources.ReadFile(filePath)
					if err != nil {
						return err
					}
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

	refStrCss := ""
	for _, cssFile := range refsCss {
		refStrCss += "\t\t<link rel=\"stylesheet\" href=\"" + cssFile + "\">\n"
	}

	refStrJs := ""
	for _, jsFile := range refsJs {
		refStrJs += "\t\t<script type=\"application/javascript\" src=\"" + jsFile + "\"></script>\n"
	}

	indexHtml = strings.ReplaceAll(indexHtml, PlaceholderCss, refStrCss)
	indexHtml = strings.ReplaceAll(indexHtml, PlaceholderJs, refStrJs)

	return &StaticHandler{
		indexHtml:   indexHtml,
		cachedFiles: cachedFiles,
		devFiles:    devFiles,
	}, nil

}
