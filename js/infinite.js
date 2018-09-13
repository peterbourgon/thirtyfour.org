function getNextPageURL(page) {
    if (page == null) {
        return null;
    }
    link = page.querySelector('a.next');
    if (link == null) {
        return null;
    }
    return link.getAttribute('href');
}

var nextPageURL = getNextPageURL(window.top.document);

function scrollPercent() {
    var h = document.documentElement,
        b = document.body,
        st = 'scrollTop',
        sh = 'scrollHeight';
    return (h[st] || b[st]) / ((h[sh] || b[sh]) - h.clientHeight) * 100;
}

var alreadyLoading = false;

window.onscroll = function () {
    if (scrollPercent() > 90 && nextPageURL && !alreadyLoading) {
        getNext(nextPageURL);
    }
}

function getNext(url) {
    var req = new XMLHttpRequest();
    req.onreadystatechange = function () {
        if (req.readyState == XMLHttpRequest.DONE) {
            if (req.status == 200) {
                res = document.createElement('html');
                res.innerHTML = req.responseText;
                images = res.querySelectorAll('a.image').forEach(function(element) {
                    document.querySelector('div.content').appendChild(element);
                });
                nextPageURL = getNextPageURL(res)
                if (!nextPageURL) {
                    window.top.document.querySelector('div.footer').style.display = 'none';
                }
                alreadyLoading = false;
            }
        }
    };

    alreadyLoading = true;
    req.open("GET", url, true);
    req.send();
}