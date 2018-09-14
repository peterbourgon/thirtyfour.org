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

set extension (echo $image | string split '.' | tail -n1)
set lastfile (ls -c1 img/ | tail -n1)
set laststr (echo $lastfile | string split '.' | head -n1)
set nextint (math $laststr + 1)
set nextstr (printf '%04d' $nextint)
set nextfile {$nextstr}.{$extension}

cp $image img/$nextfile
git add img/$nextfile
git commit -m $nextstr
git push origin master
