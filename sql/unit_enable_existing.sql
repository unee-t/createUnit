# For any question about this script, ask Franck
#
# Pre-requisite:
#	- The unit must already exists in the BZ table
#
# This script will
#	- Enable the unit
#	- Log what it does in the BZ database
#	- Exit with no error if everything went as expected
#
#################################################################
#
# UPDATE THE BELOW VARIABLES ACCORDING TO YOUR NEEDS
#
#################################################################

# We need the BZ product id for the unit
	SET @product_id = '%s';

# We need the BZ user id of the user who is making the change
    SET @bz_user_id = 'enter_the_bz_user_id_of_the_user_who_initiate_the_change';

# We need the environment you are in
#	- 1 is for DEV/Staging
#	- 2 is PROD
#	- 3 is Demo/Local installation
	SET @environment = %d;

# We also need to know the date when the unit was disabled in the MEFE
	SET @active_when = NOW();

#
########################################################################
#
# ALL THE VARIABLES WE NEED HAVE BEEN DEFINED, WE CAN RUN THE SCRIPT
#
########################################################################

# We have everything, we can now call the procedure which disables a unit
	CALL `unit_enable_existing`;


