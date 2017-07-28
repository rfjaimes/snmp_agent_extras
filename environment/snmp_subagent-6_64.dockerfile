#
#   This dockerfile is free software: you can redistribute it and/or modify
#   it under the terms of the GNU General Public License as published by
#   the Free Software Foundation, either version 3 of the License, or
#   (at your option) any later version.
#
#   This dockerfile is distributed in the hope that it will be useful,
#   but WITHOUT ANY WARRANTY; without even the implied warranty of
#   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#   GNU General Public License for more details.
#
#   DockerName: registry.intraway.com/sentinel/snmp_subagent-6_64
#  Version/Tag: @version@
#        Autor: Nicolás Dascanio
#        eMail: nicolas.dascanio@intraway.com
#          Web: www.intraway.com
#   Repository: http://gitlab.intraway.com/...
#       Issues: http://gitlab.intraway.com/.../issues
#
#         Note:
#

# Set the base image
FROM registry.intraway.com/services/puppet-6_64

################## BEGIN INSTALLATION ######################
# Correr puppet

ADD puppet /etc/puppetlabs/code
RUN /opt/iway/puppet.sh -e ci
ADD extras/init.sh /app/

ENTRYPOINT ["/app/init.sh"]
