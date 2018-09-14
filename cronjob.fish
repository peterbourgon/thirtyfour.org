#!/usr/bin/env fish

if test (count $argv) -ne 1
	echo usage: (status -f) [path]
	exit
end

set path $argv[1]

if test ! -d $path
	echo $path: not a directory
	exit
end

set now (date -u +%s)
set images (find $path -type f)
set selected ""

for f in $images
	set str (basename $f | string split '.' | head -n1)
	set unix (date -j -u -f '%Y%m%d%H%M%S' $str +%s)
	if test $unix -lt $now
		./publish_image.fish $f
		rm $f
	end
end
