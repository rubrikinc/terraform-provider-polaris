default: testacc

.PHONY: testacc
testacc:
	TF_ACC=1 go test -count=1 -timeout=120m -v $(TESTARGS) ./...
