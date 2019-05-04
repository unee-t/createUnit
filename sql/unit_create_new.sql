# For any question about this script, ask Franck
#
# Pre-requisite:
#	- The table `ut_data_to_create_units` has been updated with the information needed to create the unit
#
# This script will 
#	- Create a new unit and all the objects needed
#	- Return the bz product id for the newly created unit.
#
#################################################################
#
# UPDATE THE BELOW VARIABLES ACCORDING TO YOUR NEEDS
#
#################################################################
#
# The unit: What is the id of the unit in the table 'ut_data_to_create_units'
	SET @mefe_unit_id = '%s';
	SET @mefe_unit_id_int_value := %d;
#
# Environment: Which environment are you creating the unit in?
#	- 1 is for the DEV/Staging
#	- 2 is for the prod environment
#	- 3 is for the Demo environment
	SET @environment = %d;
#
########################################################################
#
# ALL THE VARIABLES WE NEED HAVE BEEN DEFINED, WE CAN RUN THE SCRIPT 
#
########################################################################

# We have everything, we need. We can call the script now
	CALL `unit_create_with_dummy_users`;
