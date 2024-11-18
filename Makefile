.PHONY: debug
debug:
	docker run --network="host" --rm alpine/curl-http3 curl \
		--http3 \
		--insecure \
		--verbose https://127.0.0.1

