#!/usr/bin/env fish

if test (count $argv) -lt 2
	echo usage: (basename (status -f)) image.jpg bucket-name
	exit
end

set image $argv[1]
set bucket $argv[2]

if test ! -f $image
	echo $image: not found
	exit
end

set ext (string split . $image | tail -n1)
set thumbnail (mktemp).{$ext}
cp {$image} {$thumbnail}
sips --resampleWidth 500 {$thumbnail} >/dev/null

set r1 (head -c128 /dev/urandom | md5)
set r2 (head -c128 /dev/urandom | md5)
set unique {$r1}_{$r2}

gsutil cp -a public-read {$image} gs://{$bucket}/{$r1}_{$r2}.{$ext}
gsutil cp -a public-read {$thumbnail} gs://{$bucket}/{$r1}_{$r2}_thumb.{$ext}

echo original: https://{$bucket}.storage.googleapis.com/{$r1}_{$r2}.{$ext}
echo thumbnail: https://{$bucket}.storage.googleapis.com/{$r1}_{$r2}_thumb.{$ext}

