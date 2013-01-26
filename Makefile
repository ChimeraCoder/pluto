compile:
	go build -o build/a.out server.go conf.go

clean:
	go clean
	find build/ -name *.out -delete
