REGISTRY  ?= docker.io/germainlefebvre4
IMAGE     ?= opensp8c
TAG       ?= latest

# Port configuration for development
BACKEND_PORT  ?= 8080
FRONTEND_PORT ?= 5173

.PHONY: dev dev-backend dev-frontend build build-backend build-frontend docker-build docker-build-push

dev:
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && PORT=$(BACKEND_PORT) go run ./cmd/server

dev-frontend:
	cd frontend && VITE_API_URL=http://localhost:$(BACKEND_PORT) npm run dev -- --port $(FRONTEND_PORT)

build: build-frontend build-backend

build-backend:
	cp -r frontend/dist/. backend/ui/dist/
	cd backend && go build -o ../bin/opensp8c ./cmd/server

build-frontend:
	cd frontend && npm run build

docker-build:
	docker build -t $(REGISTRY)/$(IMAGE):$(TAG) .

docker-build-push: docker-build
	docker push $(REGISTRY)/$(IMAGE):$(TAG)
