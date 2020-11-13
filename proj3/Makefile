all: api frontend

api: api.go
	go build api.go

frontend: frontend.go
	go build frontend.go

runApi: api
	./api &

runFrontend: frontend
	./frontend &

run: runApi runFrontend

stopApi:
	- ps aux | grep "./api" | awk {'print $$2'} | head -1 | xargs kill

stopFrontend:
	- ps aux | grep "./frontend" | awk {'print $$2'} | head -1 | xargs kill

stop: stopApi stopFrontend

vegeta:
	vegeta attack -workers 50 -duration=30s -targets=target.list | tee results.bin | vegeta report

clean:
	go clean
	rm -rf results.bin

