import { api, asy } from "webglue";

let divErrors;

export default {
    title: "WebGlue test",
    error(e) {
        divErrors.append(DIV(d => {
            setTimeout(() => d.fadeOut(() => d.remove()), 2000);
        }).text(e.toString()));
    },
    async render(container, page, params) {
        return [
            DIV("errors", d => divErrors = d),
            DIV("test div", [
                DIV("label").text("Error handling test"),
                DIV("line", () => {
                    let inA, inB, divResult;
                    return [
                        NUMBER(d => inA = d).val(10),
                        DIV().text("/"),
                        NUMBER(d => inB = d).val(0),
                        DIV().text("="),
                        DIV(d => divResult = d),
                        DIV("end", [
                            BUTTON().text("test").click(() => {
                                asy(async () => {
                                    let [result, reminder] = await api.div(
                                        Number.parseInt(inA.val()),
                                        Number.parseInt(inB.val())
                                    );
                                    divResult.text(result + (reminder ? " rem. " + reminder : ""));
                                })
                            })
                        ])
                    ]
                }),
                DIV("notes").text("This is a call to div API service, to demonstrate error handling.")
            ]),
            DIV("test greetings", () => {
                let inFirstName, inLastName, divGreetings;
                return [
                    DIV("label").text("Complex parameters test"),
                    DIV("line", [
                        DIV().text("First name:"),
                        TEXT(d => inFirstName = d).val("Zaphod"),
                        DIV().text("Last name:"),
                        TEXT(d => inLastName = d).val("Beeblebrox"),
                        DIV("end", [
                            BUTTON().text("test").click(() => {
                                asy(async () => {
                                    let greetings = await api.greet({
                                        firstName: inFirstName.val(),
                                        lastName: inLastName.val()
                                    });
                                    divGreetings.append(greetings.map(g => DIV().text(g)))
                                })
                            })
                        ])
                    ]
                    ),
                    DIV("greetings", d => divGreetings = d),
                    DIV("notes").text("Service parameters and reurn values may be arrays or objects. If the go function returns more than one result, it is returned as array.")
                ]
            }),
            DIV("test sessionid", [
                DIV("label").text("Session ID"),
                DIV("line", [
                    DIV(async d => {
                        d.text(await api.getId());
                    }),
                ]),
                DIV("notes").text("The getId function returns the session ID. The session ID is stored in the browser's local storage, so it is preserved even if you refresh the page.")
            ]),
            DIV("test div", [
                DIV("label").text("Server session test"),
                DIV("line", () => {
                    let divCounter;
                    return [
                        DIV().text("counter:"),
                        DIV(d => {
                            divCounter = d;
                            asy(async () => {
                                divCounter.text(await api.inc(0));
                            })
                        }),
                        DIV("end", [
                            BUTTON().text("test").click(() => {
                                asy(async () => {
                                    divCounter.text(await api.inc(1));
                                })
                            })
                        ])
                    ]
                }),
                DIV("notes").text("The inc function increments a counter in the server session. The counter is stored in the session, so it is incremented even if you refresh the page.")
            ]),
            DIV("test div", [
                DIV("label").text("Page navigation"),
                DIV("line", [
                    AHREF({ href: "/page2" }).text("Go to page 2"),
                ]),
                DIV("notes").text("This shows page navigation support. The link above takes you to another page. Note the page is not actually reloaded, it is just re-rendered.")
            ])
        ];
    }
}