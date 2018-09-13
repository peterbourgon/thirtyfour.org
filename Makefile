.PHONY: all
all: public/index.html public/css/base.css

public/index.html: index.template
	go version
	cp $< $@

public/css/base.css: base.css
	cp $< $@
