compile:
	go build -o build/pluto server.go conf.go mongo.go views.go planet_item.go

clean:
	go clean
	find build/ -name pluto -delete
