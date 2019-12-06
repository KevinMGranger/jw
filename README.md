# jw

Watch / print jenkins job output. Works on in-progress or finished jobs.

# Usage

1. Get your Jenkins User ID (not what you think it is!)
   1. Click on your name
   2. Copy the "Jenkins User ID"
2. Get your jenkins API key from jenkins settings
   1. Click on your name
   2. Click configure on the left
   3. Click "Add new token" in the API Token section
   4. Save the token somewhere safe
3. Set JENKINS_KEY to your api key, and JENKINS_USER to your user ID
4. Copy the job URL
5. Run the command as so: `jw $JOB_URL`
   1. You may need to use the `-k` flag if your certificate isn't in the system store