module.exports = function(pageNode, rootNode) {
    return new Promise((resolve, reject) => {
        resolve({
            actTime: new Date(),
            pageConfig: require('./page.json')
        });
    });
};
