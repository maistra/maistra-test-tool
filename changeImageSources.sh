#!/bin/bash
#########################################
# Change from all instances of images in imageSource list from column 1 -> 2
#########################################


NUM_IMAGES=$( cat imageSourceList | wc -l)
for (( i=1; i<=$NUM_IMAGES; i++ ))
do 
				KEY=$(sed -n "$i"p imageSourceList | awk '{print $1}' )
				VALUE=$(sed -n "$i"p imageSourceList | awk '{print $2}' )

				#echo $KEY
				#echo $VALUE

				#All image references will be yaml "image: *" or kubectl "image=*"
				#grep -r "image: $VALUE" ./*/*
				#grep -r "image=$VALUE" ./*/*


				grep -rl "image: \"$KEY\"" . |  xargs sed -i "s|$KEY|$VALUE|g"
				grep -rl "image: $KEY" . |  xargs sed -i "s|$KEY|$VALUE|g"
				grep -rl "image=$KEY" . |  xargs sed -i "s|$KEY|$VALUE|g"


done
