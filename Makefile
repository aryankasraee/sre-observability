PROMTOOL = docker run --rm -v "$(CURDIR)/prometheus:/p" --entrypoint promtool prom/prometheus:v3.15.0

.PHONY: rules test up drill down
rules:
	python3 scripts/gen-slo-rules.py --profile prod > prometheus/rules/slo.prod.rules.yml
	@mkdir -p prometheus/active
	python3 scripts/gen-slo-rules.py --profile lab > prometheus/active/slo.rules.yml
test: rules
	git diff --exit-code prometheus/rules/slo.prod.rules.yml
	$(PROMTOOL) check rules /p/rules/slo.prod.rules.yml
	$(PROMTOOL) check rules /p/active/slo.rules.yml
	$(PROMTOOL) test rules /p/rules/tests/slo_test.yml
	cd app && go vet ./... && go test ./...
up: rules
	docker compose up -d --build
drill:
	scripts/chaos-drill.sh
down:
	docker compose down -v
