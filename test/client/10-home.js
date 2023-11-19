wg.pages.home = {
    title: "WebGlue test",
    async render(container, page, params) {

        let divA, divB, divResult, divFirstName, divLastName, divGreetings, divCounter, divErrors;

        wg.showError = function (e) {
            divErrors.append(DIV(d => {
                setTimeout(() => d.fadeOut(() => d.remove()), 2000);
            }).text(e.toString()));
        }

        container.append(
            DIV("errors", d => divErrors = d),
            DIV("test div", [
                DIV("label").text("Error handling test"),
                DIV("line", [
                    INPUT(d => divA = d).val(10),
                    DIV().text("/"),
                    INPUT(d => divB = d).val(0),
                    DIV().text("="),
                    DIV(d => divResult = d),
                    DIV("end", [
                        BUTTON().text("test").click(() => {
                            wg.doAsync(async () => {
                                let [result, reminder] = await wg.api.div(
                                    Number.parseInt(divA.val()),
                                    Number.parseInt(divB.val())
                                );
                                divResult.text(result + (reminder ? " rem. " + reminder : ""));
                            })
                        })
                    ])
                ]),
                DIV("notes").text("This is a call to div API service, to demonstrate error handling.")
            ]),
            DIV("test greetings", [
                DIV("label").text("Complex parameters test"),
                DIV("line", [
                    DIV().text("First name:"),
                    INPUT(d => divFirstName = d).val("Zaphod"),
                    DIV().text("Last name:"),
                    INPUT(d => divLastName = d).val("Beeblebrox"),
                    DIV("end", [
                        BUTTON().text("test").click(() => {
                            wg.doAsync(async () => {
                                let greetings = await wg.api.greet({
                                    firstName: divFirstName.val(),
                                    lastName: divLastName.val()
                                });
                                divGreetings.append(greetings.map(g => DIV().text(g)))
                            })
                        })
                    ])
                ]),
                DIV("greetings", d => divGreetings = d),
                DIV("notes").text("Service parameters and reurn values may be arrays or objects. If the go function returns more than one result, it is returned as array.")
            ]),
            DIV("test sessionid", [
                DIV("label").text("Session ID"),
                DIV("line", [
                    DIV(async d => {
                        d.text(await wg.api.getId());
                    }),
                ]),
                DIV("notes").text("The getId function returns the session ID. The session ID is stored in the browser's local storage, so it is preserved even if you refresh the page.")
            ]),
            DIV("test div", [
                DIV("label").text("Server session test"),
                DIV("line", [
                    DIV().text("counter:"),
                    DIV(d => {
                        divCounter = d;
                        wg.doAsync(async () => {
                            divCounter.text(await wg.api.inc(0));
                        })
                    }),
                    DIV("end", [
                        BUTTON().text("test").click(() => {
                            wg.doAsync(async () => {
                                divCounter.text(await wg.api.inc(1));
                            })
                        })
                    ])
                ]),
                DIV("notes").text("The inc function increments a counter in the server session. The counter is stored in the session, so it is incremented even if you refresh the page.")
            ]),
            DIV("test div", [
                DIV("label").text("Page navigation"),
                DIV("line", [
                    AHREF({href: "/page2"}).text("Go to page 2"),
                ]),
                DIV("notes").text("This shows page navigation support. The link above takes you to another page. Note the page is not actually reloaded, it is just re-rendered.")
            ])
       )
    }
}