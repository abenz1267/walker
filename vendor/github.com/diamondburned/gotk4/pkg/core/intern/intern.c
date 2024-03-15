#include "intern.h"

const gchar *gotk4_object_type_name(gpointer obj) {
  return G_OBJECT_TYPE_NAME(obj);
};

gboolean gotk4_intern_remove_toggle_ref(gpointer obj) {
  // First, remove the toggle reference. This forces the object to be freed,
  // calling any necessary finalizers.
  g_object_remove_toggle_ref(G_OBJECT(obj), (GToggleNotify)goToggleNotify,
                             NULL);

  // Only once the object is freed, we can remove it from the weak reference
  // registry, since now the finalizers will not be called anymore.
  goFinishRemovingToggleRef(obj);

  return FALSE;
}
