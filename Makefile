all:
	go build -o ./btexamples ./btexamples/*.go && \
	pushd btexamples && ( ./btexamples || popd )
