BINARY      := benefitshow
PKG         := ./cmd/benefitshow

GCP_PROJECT := band-benefit-voting-app
GCP_ZONE    := us-central1-f
GCP_VM      := instance-20260501-161311
REMOTE_DIR  := /opt/benefitshow

GCLOUD_FLAGS := --zone=$(GCP_ZONE) --project=$(GCP_PROJECT)

.PHONY: build run lint clean build-linux deploy reset-db

build:
	go build -o $(BINARY) $(PKG)

run: build
	@set -a; [ -f .env ] && . ./.env; set +a; ./$(BINARY) serve

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || { echo "gofmt: files need formatting:"; gofmt -l .; exit 1; }

clean:
	rm -f $(BINARY) $(BINARY)-linux

reset-db:
	rm -f benefitshow.db benefitshow.db-shm benefitshow.db-wal

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BINARY)-linux $(PKG)

deploy: build-linux
	gcloud compute scp $(BINARY)-linux $(GCP_VM):$(REMOTE_DIR)/$(BINARY).new $(GCLOUD_FLAGS)
	gcloud compute ssh $(GCP_VM) $(GCLOUD_FLAGS) --command='install -m 0755 $(REMOTE_DIR)/$(BINARY).new $(REMOTE_DIR)/$(BINARY) && rm $(REMOTE_DIR)/$(BINARY).new && sudo systemctl restart $(BINARY)'
