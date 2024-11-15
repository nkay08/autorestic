# Depends On

If multiple location shall be backed up in a particular order, you can use the `depends_on` option. 
Multiple locations can be specified.  
However, you should take care not to introduce cyclic dependencies.  
If backing up only a selected list of locations, locations that are specified in `depends_on` will not get added if they are not in the list of selected locations.

A well-defined order of execution can be useful in conjunction with `hooks`.

###### Example

```yaml | .autorestic.yml
locations:
  location_A:
    from: /data_A
    to: 
    - backend_A
    depends_on:
    - location_C
  location_B:
    from: /data_B
    to: 
    - backend_B
    depends_on:
    - location_A
    - location_C
  location_C:
    from: /data_C
    to: 
    - backend_C
```

will result in the order of execution of backups `location_C` -> `location_A` ->  `location_B`.