#include <glib-object.h>

extern void goToggleNotify(gpointer, GObject *, gboolean);
extern void goFinishRemovingToggleRef(gpointer);
const gchar *gotk4_object_type_name(gpointer obj);
gboolean gotk4_intern_remove_toggle_ref(gpointer obj);
