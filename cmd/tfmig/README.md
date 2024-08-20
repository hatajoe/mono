# tfmig

tfmig is the tool to migrate terraform state between project.  
you can move one or more number of states that you want to move at once.  
you can select states by fazzyfinder CUI interface.  

# usage

## basics

you can move multiple state between two project, e.g, A to B by following command: 

```
% tfmig -s /path/to/A -d /path/to/B [-w workspace-name]
```

- s: path to terraform project that you wanna move state from
- d: path to terraform project that you wanna move state into
- w (optional): workspace name

## rollback

tfmig is going to make backup file of the state to .bak dir in each project dir.  
you can rollback states that you have moved by following command:

```
% cd /path/to/A
% terraform state push -force .bak/2021-11-05-32-27-09.tfstate
% rm -rf .bak
% cd /path/to/B
% terraform state push -force .bak/2021-11-05-32-27-09.tfstate
% rm -rf .bak
```

# licence

MIT
