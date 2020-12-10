all: backend frontend

backend: backend.go
	go build backend.go

frontend: frontend.go
	go build frontend.go

run-backend: backend
	./api &

run-frontend: frontend
	./frontend &

run: run-backend run-frontend

stop-backend:
	- ps aux | grep "./backend" | awk {'print $$2'} | head -1 | xargs kill

stop-frontend:
	- ps aux | grep "./frontend" | awk {'print $$2'} | head -1 | xargs kill

stop: stop-frontend stop-backend

vegeta:
	vegeta attack -workers 50 -duration=30s -targets=target.list | tee results.bin | vegeta report

clean:
	go clean
	rm -rf results.bin

