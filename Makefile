.PHONY: all
all: public/css/base.css public/index.html

public/index.html: index.template public/img
	go run src/render/*.go -v -template-file index.template -image-path public/img -per-page 3 -output-path public

public/css/base.css: css/base.css
	mkdir -p $(shell dirname $@)
	cp $< $@

public/img: img
	mkdir -p $@
	cp $</* $@/
