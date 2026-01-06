#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "backup | build | clear | db | img";
	@echo "macro: everything";

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

backup:
	@echo "Making backup ..."
	@rclone copy -P --exclude '.git/**' --exclude 'cache/' . gsc:koopi/

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
	@cd markets && find . -type f \( -name "*.jpg" -o -name "*.png" \) -print0 | xargs -0 -P $(shell nproc) -I {} sh -c ' \
		INPUT="$$1"; \
		OUTPUT=$${INPUT%.*}.webp; \
		if [ "$$INPUT" -nt "$$OUTPUT" ] || [ ! -f "$$OUTPUT" ]; then \
			convert "$$INPUT" -quality 80 "$$OUTPUT"; \
		fi \
	' _ {}

# macros
everything: clear db img
	@-git add -A
	@-git commit -am 'automatic update'

cf:
	@mkdir -p export/images export/markets
	@rsync -av --delete export-template/ export/
	@rsync -av --delete images/ export/images/
	@rsync -av --delete markets/ export/markets/
	@cd export && git add -A
	@cd export && git commit -m 'automatic update: $$(date)' || true
	@cd export && git push origin master
