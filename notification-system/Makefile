.PHONY: up down producer sms-worker email-worker

up:
	docker compose up -d

down:
	docker compose down -v

producer:
	cd producer && go run .

sms-worker:
	cd sms-worker && go run .

email-worker:
	cd email-worker && deno run dev
