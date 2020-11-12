all: api frontend

api: api.go
	go build api.go

frontend: frontend.go
	go build frontend.go

run: api frontend
	./api &
	./frontend &

stop:
	ps aux | grep "./api" | awk {'print $$2'} | head -1 | xargs kill
	ps aux | grep "./frontend" | awk {'print $$2'} | head -1 | xargs kill
	
clean:
	go clean

