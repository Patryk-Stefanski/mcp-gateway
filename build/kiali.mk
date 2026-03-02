# Kiali Service Mesh Observability

KIALI_VERSION ?= 2.3
KIALI_NAMESPACE = istio-system
HELM ?= bin/helm

.PHONY: kiali-install-impl
kiali-install-impl: $(HELM)
	@echo "Installing Kiali Operator via Helm..."
	@-$(HELM) repo add kiali https://kiali.org/helm-charts 2>/dev/null
	@$(HELM) repo update
	@if $(HELM) list -n $(KIALI_NAMESPACE) | grep -q kiali-operator; then \
		echo "Kiali operator already installed, upgrading..."; \
		$(HELM) upgrade \
			kiali-operator kiali/kiali-operator \
			--namespace $(KIALI_NAMESPACE) \
			--version $(KIALI_VERSION) \
			--set cr.create=true \
			--set cr.namespace=$(KIALI_NAMESPACE) \
			--set cr.spec.auth.strategy=anonymous \
			--set cr.spec.deployment.cluster_wide_access=true \
			--set cr.spec.external_services.prometheus.url=http://prometheus.observability:9090 \
			--set cr.spec.external_services.grafana.enabled=true \
			--set cr.spec.external_services.grafana.url=http://localhost:3000 \
			--set cr.spec.external_services.grafana.in_cluster_url=http://grafana.observability:3000 \
			--set cr.spec.external_services.tracing.enabled=true \
			--set cr.spec.external_services.tracing.provider=tempo \
			--set cr.spec.external_services.tracing.url=http://localhost:3000/explore \
			--set cr.spec.external_services.tracing.in_cluster_url=http://tempo.observability:3200 \
			--set cr.spec.external_services.tracing.use_grpc=false \
			--set cr.spec.external_services.tracing.tempo_config.org_id=1 \
			--set cr.spec.external_services.tracing.tempo_config.datasource_uid=tempo \
			--set cr.spec.external_services.tracing.tempo_config.url_format=grafana \
			--set cr.spec.external_services.tracing.namespace_selector=true \
			--set "cr.spec.external_services.istio.gateway_api_classes[0].class_name=istio" \
			--wait \
			--timeout=300s; \
	else \
		$(HELM) install \
			kiali-operator kiali/kiali-operator \
			--namespace $(KIALI_NAMESPACE) \
			--create-namespace \
			--version $(KIALI_VERSION) \
			--set cr.create=true \
			--set cr.namespace=$(KIALI_NAMESPACE) \
			--set cr.spec.auth.strategy=anonymous \
			--set cr.spec.deployment.cluster_wide_access=true \
			--set cr.spec.external_services.prometheus.url=http://prometheus.observability:9090 \
			--set cr.spec.external_services.grafana.enabled=true \
			--set cr.spec.external_services.grafana.url=http://localhost:3000 \
			--set cr.spec.external_services.grafana.in_cluster_url=http://grafana.observability:3000 \
			--set cr.spec.external_services.tracing.enabled=true \
			--set cr.spec.external_services.tracing.provider=tempo \
			--set cr.spec.external_services.tracing.url=http://localhost:3000/explore \
			--set cr.spec.external_services.tracing.in_cluster_url=http://tempo.observability:3200 \
			--set cr.spec.external_services.tracing.use_grpc=false \
			--set cr.spec.external_services.tracing.tempo_config.org_id=1 \
			--set cr.spec.external_services.tracing.tempo_config.datasource_uid=tempo \
			--set cr.spec.external_services.tracing.tempo_config.url_format=grafana \
			--set cr.spec.external_services.tracing.namespace_selector=true \
			--set "cr.spec.external_services.istio.gateway_api_classes[0].class_name=istio" \
			--wait \
			--timeout=300s; \
	fi
	@echo ""
	@echo "Kiali installed. Access via: make kiali-forward"

.PHONY: kiali-uninstall-impl
kiali-uninstall-impl: $(HELM)
	@echo "Uninstalling Kiali..."
	@-$(HELM) uninstall kiali-operator --namespace $(KIALI_NAMESPACE)
	@-kubectl delete kiali kiali --namespace $(KIALI_NAMESPACE) 2>/dev/null || true
	@echo "Kiali uninstalled."

.PHONY: kiali-forward-impl
kiali-forward-impl:
	@echo "Kiali: http://localhost:20001"
	@kubectl port-forward svc/kiali -n $(KIALI_NAMESPACE) 20001:20001
