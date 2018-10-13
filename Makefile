.PHONY: all
all: public/favicon.ico public/css/base.css public/js/infinite.js public/index.html
	@echo all OK

public/favicon.ico: favicon.ico
	mkdir -p $(@D)
	cp $< $@

public/css/base.css: css/base.css
	mkdir -p $(@D)
	cp $< $@

public/js/infinite.js: js/infinite.js
	mkdir -p $(@D)
	cp $< $@

public/index.html: index.template public/img src/render/*.go
	go run src/render/*.go -v -template-file index.template -image-path public/img -per-page 10 -output-path public

public/img: img
	mkdir -p $@
	cp $</* $@/

.PHONY: clean
clean:
	rm -rf public
