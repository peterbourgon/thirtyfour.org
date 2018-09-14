#!/usr/bin/env fish

if test (count $argv) -ne 1
	echo usage: (status -f) [path]
	exit
end

set path $argv[1]
set images (find $path)
set image ""

while true
	set index (math (random) '%' (count $images))
	set image $images[$index]
	echo $image | pbcopy
	open $image
	read -P "This one? [yN] " -n 1 input
	if test $input = "y"
		break
	end
end

./publish_image.fish $image
