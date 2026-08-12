package email

import "fmt"

// catalog holds every string that appears in an email, per locale. Values with
// %s placeholders are documented at their use site in booking.go; the argument
// order is part of the key's contract and must match across locales.
//
// catalog_test.go asserts every locale carries the exact same key set, so a
// half-translated email can't ship.
var catalog = map[Locale]map[string]string{
	LocaleFR: {
		"brand.tagline":  "Deux adresses familiales, Los Alcázares",
		"footer.signed":  "Delphine & Christophe",
		"footer.role":    "Vos hôtes",
		"footer.reply":   "Vous pouvez répondre directement à cet e-mail.",
		"label.villa":    "Logement",
		"label.checkin":  "Arrivée",
		"label.checkout": "Départ",
		"label.nights":   "Nuits",
		"label.guests":   "Voyageurs",
		"label.name":     "Nom",
		"label.email":    "E-mail",
		"label.phone":    "Téléphone",
		"label.message":  "Message",
		"label.ref":      "Référence",

		"greeting":           "Bonjour %s,",
		"guests.adult.one":   "%d adulte",
		"guests.adult.other": "%d adultes",
		"guests.child.one":   "%d enfant",
		"guests.child.other": "%d enfants",

		// %s = villa name
		"received.subject": "Votre demande de réservation — %s",
		"received.heading": "Demande reçue",
		"received.p1":      "Nous avons bien reçu votre demande de séjour à %s. Elle n'est pas encore confirmée : nous vérifions les disponibilités et nous revenons vers vous très vite.",
		"received.p2":      "Si vous souhaitez nous préciser quelque chose, répondez simplement à cet e-mail.",
		"received.note":    "Aucun montant ne vous est débité à ce stade.",

		// %s = villa name, %s = check-in, %s = check-out
		"approved.subject": "Votre séjour à %s est confirmé",
		"approved.heading": "Séjour confirmé",
		"approved.p1":      "Bonne nouvelle : votre séjour à %s du %s au %s est confirmé.",
		"approved.p2":      "Nous vous écrirons quelques jours avant votre arrivée avec les informations pratiques : accès au logement, horaires d'arrivée et contacts sur place.",
		"approved.note":    "Un empêchement, une question ? Écrivez-nous, nous trouverons une solution.",

		"rejected.subject": "Votre demande pour %s",
		"rejected.heading": "Demande non retenue",
		"rejected.p1":      "Merci de l'intérêt que vous portez à %s. Nous ne pouvons malheureusement pas accueillir votre demande pour les dates du %s au %s.",
		"rejected.p2":      "D'autres dates sont peut-être libres, et notre second logement l'est parfois quand le premier ne l'est pas. Répondez à cet e-mail et nous chercherons avec vous.",

		"cancelled.subject": "Votre réservation à %s a été annulée",
		"cancelled.heading": "Réservation annulée",
		"cancelled.p1":      "Votre réservation à %s du %s au %s a été annulée.",
		"cancelled.p2":      "S'il s'agit d'une erreur, répondez à cet e-mail : nous la rétablirons si les dates sont encore libres.",

		// %s = villa name, %s = check-in, %s = check-out
		"owner.subject":   "Nouvelle demande — %s · %s → %s",
		"owner.heading":   "Nouvelle demande de réservation",
		"owner.p1":        "Une nouvelle demande vient d'arriver depuis le site. Elle est en attente de votre réponse dans le back-office.",
		"owner.p2":        "Répondez à cet e-mail pour écrire directement au voyageur.",
		"owner.nomessage": "(aucun message)",
	},
	LocaleEN: {
		"brand.tagline":  "Two family homes, Los Alcázares",
		"footer.signed":  "Delphine & Christophe",
		"footer.role":    "Your hosts",
		"footer.reply":   "You can reply directly to this email.",
		"label.villa":    "Property",
		"label.checkin":  "Check-in",
		"label.checkout": "Check-out",
		"label.nights":   "Nights",
		"label.guests":   "Guests",
		"label.name":     "Name",
		"label.email":    "Email",
		"label.phone":    "Phone",
		"label.message":  "Message",
		"label.ref":      "Reference",

		"greeting":           "Hello %s,",
		"guests.adult.one":   "%d adult",
		"guests.adult.other": "%d adults",
		"guests.child.one":   "%d child",
		"guests.child.other": "%d children",

		"received.subject": "Your booking request — %s",
		"received.heading": "Request received",
		"received.p1":      "We've received your request to stay at %s. It isn't confirmed yet: we're checking availability and will get back to you very soon.",
		"received.p2":      "If there's anything you'd like to tell us, simply reply to this email.",
		"received.note":    "Nothing is charged at this stage.",

		"approved.subject": "Your stay at %s is confirmed",
		"approved.heading": "Stay confirmed",
		"approved.p1":      "Good news: your stay at %s from %s to %s is confirmed.",
		"approved.p2":      "We'll write to you a few days before you arrive with the practical details: how to get in, arrival times and who to contact locally.",
		"approved.note":    "Plans changed, or a question? Write to us and we'll find a way.",

		"rejected.subject": "Your request for %s",
		"rejected.heading": "Request not accepted",
		"rejected.p1":      "Thank you for your interest in %s. Unfortunately we can't take your request for %s to %s.",
		"rejected.p2":      "Other dates may be free, and our second property is sometimes available when the first one isn't. Reply to this email and we'll look with you.",

		"cancelled.subject": "Your booking at %s has been cancelled",
		"cancelled.heading": "Booking cancelled",
		"cancelled.p1":      "Your booking at %s from %s to %s has been cancelled.",
		"cancelled.p2":      "If that's a mistake, reply to this email: we'll reinstate it if the dates are still free.",

		"owner.subject":   "New request — %s · %s → %s",
		"owner.heading":   "New booking request",
		"owner.p1":        "A new request just came in from the website. It's waiting for your answer in the back-office.",
		"owner.p2":        "Reply to this email to write to the guest directly.",
		"owner.nomessage": "(no message)",
	},
	LocaleES: {
		"brand.tagline":  "Dos casas familiares, Los Alcázares",
		"footer.signed":  "Delphine & Christophe",
		"footer.role":    "Sus anfitriones",
		"footer.reply":   "Puede responder directamente a este correo.",
		"label.villa":    "Alojamiento",
		"label.checkin":  "Llegada",
		"label.checkout": "Salida",
		"label.nights":   "Noches",
		"label.guests":   "Huéspedes",
		"label.name":     "Nombre",
		"label.email":    "Correo",
		"label.phone":    "Teléfono",
		"label.message":  "Mensaje",
		"label.ref":      "Referencia",

		"greeting":           "Hola %s:",
		"guests.adult.one":   "%d adulto",
		"guests.adult.other": "%d adultos",
		"guests.child.one":   "%d niño",
		"guests.child.other": "%d niños",

		"received.subject": "Su solicitud de reserva — %s",
		"received.heading": "Solicitud recibida",
		"received.p1":      "Hemos recibido su solicitud de estancia en %s. Todavía no está confirmada: estamos comprobando la disponibilidad y le responderemos muy pronto.",
		"received.p2":      "Si desea indicarnos algo, basta con responder a este correo.",
		"received.note":    "No se le cobra ningún importe en esta fase.",

		"approved.subject": "Su estancia en %s está confirmada",
		"approved.heading": "Estancia confirmada",
		"approved.p1":      "Buenas noticias: su estancia en %s del %s al %s está confirmada.",
		"approved.p2":      "Le escribiremos unos días antes de su llegada con la información práctica: acceso al alojamiento, horarios de llegada y contactos en el lugar.",
		"approved.note":    "¿Un imprevisto, una duda? Escríbanos y encontraremos una solución.",

		"rejected.subject": "Su solicitud para %s",
		"rejected.heading": "Solicitud no aceptada",
		"rejected.p1":      "Gracias por su interés en %s. Lamentablemente no podemos atender su solicitud para las fechas del %s al %s.",
		"rejected.p2":      "Puede que otras fechas estén libres, y nuestro segundo alojamiento lo está a veces cuando el primero no. Responda a este correo y lo buscaremos con usted.",

		"cancelled.subject": "Su reserva en %s ha sido cancelada",
		"cancelled.heading": "Reserva cancelada",
		"cancelled.p1":      "Su reserva en %s del %s al %s ha sido cancelada.",
		"cancelled.p2":      "Si se trata de un error, responda a este correo: la restableceremos si las fechas siguen libres.",

		"owner.subject":   "Nueva solicitud — %s · %s → %s",
		"owner.heading":   "Nueva solicitud de reserva",
		"owner.p1":        "Acaba de llegar una nueva solicitud desde el sitio web. Está esperando su respuesta en el back-office.",
		"owner.p2":        "Responda a este correo para escribir directamente al huésped.",
		"owner.nomessage": "(sin mensaje)",
	},
}

// t resolves a catalog key, falling back to the default locale rather than
// rendering an empty string into an email a guest will read.
func t(loc Locale, key string) string {
	if s, ok := catalog[loc][key]; ok {
		return s
	}
	return catalog[DefaultLocale][key]
}

func tf(loc Locale, key string, args ...any) string {
	return fmt.Sprintf(t(loc, key), args...)
}

// plural picks between "<prefix>.one" and "<prefix>.other". The three locales
// this site ships all split at exactly one, so a full CLDR plural-rules
// implementation would be dead weight.
func plural(loc Locale, prefix string, n int) string {
	suffix := ".other"
	if n == 1 {
		suffix = ".one"
	}
	return tf(loc, prefix+suffix, n)
}
