var url = window.location;
var windowHref = url.href;
var path = url.pathname;

const windowLocation = (path) => {
    console.log(path);
    let results = document.getElementById("resultCount");

    if(path === "/top-headlines") {
        results.style.display = "none";
    } else {
        results.style.display = "block";
    }
};

windowLocation(path);
