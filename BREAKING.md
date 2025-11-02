## Keybinds Hints are gone

Walker now properly differentiates between global keybinds and entry-specific ones. If you are missing global keybinds, add this to your layout:

```xml
<child>
  <object class="GtkBox" id="GlobalKeybinds">
    <property name="hexpand">true</property>
    <property name="spacing">10</property>
    <property name="margin-top">10</property>
    <style>
      <class name="global-keybinds"></class>
    </style>
  </object>
</child>
```
