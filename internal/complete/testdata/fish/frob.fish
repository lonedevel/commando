# Sample in the style of fish's completion files.
complete -c frob -l color -f -a "never always auto" -d "Use colors"
complete -c frob -s X -l request -xa 'GET POST PATCH' -d 'Request method'
complete -c frob -l sort -x -d "Sort key" -a "
	size\t'Sort by size'
	extension\t'Sort by extension'
	none\tDon\'t\ sort
"
complete -c frob -l key-type -d 'Key type' -a 'PEM, DER ENG'
complete -c frob -n '__fish_seen_subcommand_from push' -l mode -a 'fast slow'
complete -c frob -l user -xa '(__fish_complete_users)'
complete -c other -l color -a 'red green'
and complete -c frob -o depth -a "1 2"
