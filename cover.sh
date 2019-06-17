mkdir -p coverHtml
go test -coverpkg=./... -coverprofile=coverHtml/coverage.data ./...
go tool cover -html=coverHtml/coverage.data -o coverHtml/coverage.html
go tool cover -func=coverHtml/coverage.data -o coverHtml/coverage.txt
/bin/bash -c 'firefox $PWD/coverHtml/coverage.html'
