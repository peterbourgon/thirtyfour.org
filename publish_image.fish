#!/usr/bin/env fish

if test (count $argv) -ne 1
	echo usage: (status -f) [filename]
	exit
end

set image $argv[1]

if test ! -f $image 
    echo $image: not a file
    exit
end

set hash (shasum --binary --algorithm 256 $image | awk '{print $1}')
for f in (find img -type f)
	if test $hash = (shasum --binary --algorithm 256 $f | awk '{print $1}')
		echo "duplicate: $f"
		exit
	end
end

set extension (echo $image | string split '.' | tail -n1)
set lastfile (ls -c1 img | sort | tail -n1)
set laststr (echo $lastfile | string split '.' | head -n1)
set nextstr (printf '%04d' (math $laststr + 1))
set nextfile {$nextstr}.{$extension}

cp $image img/$nextfile
echo git add img/$nextfile
echo git commit -m $nextstr
echo git push origin master
