export default {
    title: "WebGlue page2",
    async render(container, page, params) {
        return [
            DIV("test div", [
                DIV("label").text("Just another page"),
                DIV("line", [
                    AHREF({href: "/"}).text("Go home"),
                ]),
                DIV("notes").text("This shows page navigation support. If you come here from home page, you can go back by clicking the link above or by pressing the browser's back button.")
            ])
        ];
    }
}