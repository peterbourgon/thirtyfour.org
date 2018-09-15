.PHONY: all
all: public/favicon.ico public/css/base.css public/js/infinite.js public/index.html
	@echo all OK

public/favicon.ico: favicon.ico public
	cp $< $@

public/css/base.css: css/base.css public/css 
	cp $< $@

public/js/infinite.js: js/infinite.js public/js
	cp $< $@

public/index.html: index.template public/img src/render/*.go
	go run src/render/*.go -v -template-file index.template -image-path public/img -per-page 3 -output-path public

public/css: public
	mkdir $@

public/js: public
	mkdir $@

public/img: img public
	mkdir $@
	cp $</* $@/

public:
	mkdir public

.PHONY: clean
clean:
	rm -rf public
