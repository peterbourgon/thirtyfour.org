.PHONY: all
all: public/css/base.css public/js/infinite.js public/index.html
	@echo all OK

public/index.html: index.template public/img
	go run src/render/*.go -v -template-file index.template -image-path public/img -per-page 25 -output-path public

public/css:
	mkdir -p $@

public/css/base.css: css/base.css public/css 
	cp $< $@

public/js:
	mkdir -p $@

public/js/infinite.js: js/infinite.js public/js
	cp $< $@

public/img: img
	mkdir -p $@
	cp $</* $@/
