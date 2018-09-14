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

set now (date -u +%Y%m%d%H%M%S)
set images (find $path -type f)
set selected ""
set publish (dirname (status -f))/publish_image.fish

for f in $images
	set str (basename $f | string split '.' | head -n1)
	bash -c '[[ $now > $str ]]' # :(
	if test $status -eq 0 
		$publish $f
		rm $f
	end
end
