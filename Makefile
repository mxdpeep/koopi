#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "build | clear | db | img | everything";

clear:
	@echo "Cache cleanup ..."
	@-find ./cache/ -type f -mtime +5 -delete -print 2> /dev/null || true
	@-find ./cache/ -type f -mtime +2 2> /dev/null \
		| head -n 1000 \
		| shuf \
		| head -n 50 \
		| xargs -r -d '\n' rm -f 2> /dev/null || true

build:
	@echo "Building app ..."
	@cd go/ && go build -o koopi .

db: build
	@cd go/ && ./koopi

img:
	@echo "Converting images ..."
	@cd images && find . -type f \( -name "*.jpg" -o -name "*.png" \) -print0 | xargs -0 -P $(shell nproc) -I {} sh -c ' \
		INPUT="$$1"; \
		OUTPUT=$${INPUT%.*}.webp; \
		if [ "$$INPUT" -nt "$$OUTPUT" ] || [ ! -f "$$OUTPUT" ]; then \
			convert "$$INPUT" -quality 80 "$$OUTPUT"; \
		fi \
	' _ {}

# macro
everything: clear db img
	@-git add -A
	@-git commit -am 'automatic update'
