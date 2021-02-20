run:
	go run main/main.go
test:
	go test ./...
coverage:
	go test -failfast=true ./... -coverprofile cover.out
	go tool cover -html=cover.out
	rm cover.out
mocks:
	mockery --name=ExtRetriever --recursive=true --case=underscore --output=./pkg/testhelper/mocks;
	mockery --name=DbHandler --recursive=true --case=underscore --output=./pkg/testhelper/mocks;
	mockery --name=FileReader --recursive=true --case=underscore --output=./pkg/testhelper/mocks;
	mockery --name=HTTPClient --recursive=true --case=underscore --output=./pkg/testhelper/mocks;
