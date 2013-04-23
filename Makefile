compile:
	go build -o build/a.out server.go conf.go mongo.go views.go planet_item.go

clean:
	go clean
	find build/ -name *.out -delete
