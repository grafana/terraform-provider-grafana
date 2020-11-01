deps:
	if ! command -v certin >/dev/null; then \
		go get -u github.com/joemiller/certin ;\
	fi

generate: deps
	@certin create ca.key      ca.crt      --is-ca --cn "CA"
	@certin create grafana.key grafana.crt --signer-key ca.key --signer-cert ca.crt  --cn "grafana" --sans "localhost"
	@certin create client.key  client.crt  --signer-key ca.key --signer-cert ca.crt  --cn "client"
