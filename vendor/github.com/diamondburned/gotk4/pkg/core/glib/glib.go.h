/*
 * Copyright (c) 2013-2014 Conformal Systems <info@conformal.com>
 *
 * This file originated from: http://opensource.conformal.com/
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

#ifndef __GLIB_GO_H__
#define __GLIB_GO_H__

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include <gio/gio.h>
#define G_SETTINGS_ENABLE_BACKEND
#include <gio/gsettingsbackend.h>
#include <glib-object.h>
#include <glib.h>
#include <glib/gi18n.h>
#include <locale.h>

/* GObject Type Casting */
static GType _g_type_from_instance(gpointer instance) {
  return (G_TYPE_FROM_INSTANCE(instance));
}

static GValue *alloc_gvalue_list(int n) {
  GValue *valv;

  valv = g_new0(GValue, n);
  return (valv);
}

static void val_list_insert(GValue *valv, int i, GValue *val) {
  valv[i] = *val;
}

/*
 * GValue
 */

static GValue *_g_value_alloc() { return (g_new0(GValue, 1)); }

static GValue *_g_value_init(GType g_type) {
  GValue *value;

  value = g_new0(GValue, 1);
  return (g_value_init(value, g_type));
}

static gboolean _g_type_is_value(GType g_type) {
  return (G_TYPE_IS_VALUE(g_type));
}

static gboolean _g_is_value(GValue *val) { return (G_IS_VALUE(val)); }

static GType _g_value_type(GValue *val) { return (G_VALUE_TYPE(val)); }

static const gchar *_g_value_type_name(GValue *val) {
  return (G_VALUE_TYPE_NAME(val));
}

static GType _g_value_fundamental(GType type) {
  return (G_TYPE_FUNDAMENTAL(type));
}

static GObjectClass *_g_object_get_class(GObject *object) {
  return (G_OBJECT_GET_CLASS(object));
}

/*
 * Closure support
 */

extern void _gotk4_removeSourceFunc(gpointer data);
extern gboolean _gotk4_sourceFunc(gpointer data);

extern void _gotk4_goMarshal(GClosure *, GValue *, guint, GValue *, gpointer,
                             GObject *);
extern void _gotk4_notifyHandlerTramp(gpointer, gpointer, guintptr);

extern void _gotk4_removeClosure(GObject *, GClosure *);
extern void _gotk4_removeGeneratedClosure(guintptr, GClosure *);

static inline guint _g_signal_new(const gchar *name) {
  return g_signal_new(name, G_TYPE_OBJECT, G_SIGNAL_RUN_FIRST | G_SIGNAL_ACTION,
                      0, NULL, NULL, g_cclosure_marshal_VOID__POINTER,
                      G_TYPE_NONE, 0);
}

static void init_i18n(const char *domain, const char *dir) {
  setlocale(LC_ALL, "");
  bindtextdomain(domain, dir);
  bind_textdomain_codeset(domain, "UTF-8");
  textdomain(domain);
}

static const char *localize(const char *string) { return _(string); }

#endif
