// Package toc provides a reusable client for the TOC (Toc/Oscar) text-based
// instant messaging protocol spoken by the Open OSCAR Server (and classic AOL
// TOC gateways).
//
// The client implements the SFLAP/FLAP wire framing, the "FLAPON" sign-on
// handshake, TOC password roasting, and the common commands a bot needs
// (toc_signon, toc_init_done, toc_send_im, toc_set_away). It depends only on
// the Go standard library.
//
// Typical usage:
//
//	c, err := toc.Dial("127.0.0.1:9898", toc.Options{
//	    Handler: myHandler{},
//	})
//	if err != nil { log.Fatal(err) }
//	defer c.Close()
//	if err := c.SignIn("screenname", "password"); err != nil { log.Fatal(err) }
//	if err := c.SendIM("buddy", "hello"); err != nil { log.Fatal(err) }
//	log.Fatal(c.Receive(context.Background()))
//
// The wire format matches the server implementation in server/toc so that this
// client interoperates with Open OSCAR Server out of the box.
package toc
