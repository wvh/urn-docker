#!/usr/bin/env python

import string, re, sys, math, time, logging, logging.config, xml.sax
import multiprocessing
from urllib.request import urlopen
from datetime import datetime, timedelta
from io import StringIO
import psycopg2
from util import normalise

def err_print(s):
    print(s, file=sys.stderr)
    
class BaseFormatHandler(xml.sax.ContentHandler):
    def __init__(self, cursor, source_id, start_url, title, url_pattern, urn_type):
        xml.sax.ContentHandler.__init__(self)
        self.cursor = cursor
        self.source_id = source_id
        self.start_url = start_url
        self.title = title
        self.url_pattern = re.compile(url_pattern)
        self.urn_type = urn_type
        
    def set_url(self, url):
        if url and self.url_pattern.match(url) is not None:
            self.url = url
        else:
            log_debug('%s: rejecting URL: %s' % (self.title, url))

    def set_urn(self, urn):
        if urn:
            self.urn = urn

    def _update_history(self, urn, r_component, url_old, url_new, url_type_old, url_type_new):
        print('update_history start')
        # FIX: add harvest_time !!! 
        if url_type_old and url_type_new:
            self.cursor.execute("INSERT INTO urnhistory (urn, r_component, url_old, url_new, url_type_old, url_type_new, source_url) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s')" % (urn, r_component, url_old, url_new, url_type_old, url_type_new, self.start_url))
        elif url_type_old:
            self.cursor.execute("INSERT INTO urnhistory (urn, r_component, url_old, url_new, url_type_old, source_url) VALUES ('%s', '%s', '%s', '%s', '%s', '%s')" % (urn, r_component, url_old, url_new, url_type_old, self.start_url))
        elif url_type_new:
            self.cursor.execute("INSERT INTO urnhistory (urn, r_component, url_old, url_new, url_type_new, source_url) VALUES ('%s', '%s', '%s', '%s', '%s', '%s')" % (urn, r_component, url_old, url_new, url_type_new, self.start_url))
        else:
            self.cursor.execute("INSERT INTO urnhistory (urn, r_component, url_old, url_new, source_url) VALUES ('%s', '%s', '%s', '%s', '%s')" % (urn, r_component, url_old, url_new, self.start_url))
        print('update_history end')
            
    def write_url(self):
        #urn = self.urn.strip().lower()
        urn = normalise(self.urn.strip())
        if urn == None:
            print('Invalid URN: %s' % self.urn)
            return
        #self.cursor.execute("SELECT (urn, url, source_id, type) FROM urn2url WHERE urn = '%s'" % urn)
        self.cursor.execute("SELECT * FROM urn2url WHERE urn = '%s'" % urn)
        existing_rows = [t for t in self.cursor.fetchall()]
        existing_urns = [row[0] for row in existing_rows]
        existing_source_ids = [row[2] for row in existing_rows]
        if urn in existing_urns:
            if self.source_id in existing_source_ids:
                for row in existing_rows:
                    if row[2] == self.source_id:
                        if row[1] != self.url:
                            # The URL for this URN in this source has changed!
                            print('< urn change')
                            self.cursor.execute("UPDATE urn2url SET url='%s' WHERE urn='%s' AND source_id = '%s'" % (self.url, urn, self.source_id))
                            self._update_history(urn=urn, r_component=None,
                                                 url_old=row[1], url_new=self.url,
                                                 url_type_old=None, url_type_new=self.urn_type)
                            print('update')
            else:
                print('< (another) urn from new source')
                # We have already harvested URN from some other source.
                self.cursor.execute("INSERT INTO urn2url (urn, url, source_id, url_type) VALUES ('%s', '%s', '%s', '%s')" % (urn, self.url, self.source_id, self.urn_type))
                self._update_history(urn=urn, r_component=None,
                                     url_old=None, url_new=self.url,
                                     url_type_old=None, url_type_new=self.urn_type)
                print('insert another')
        else:
            # New URN!
            print('< new urn')
            self.cursor.execute("INSERT INTO urn2url (urn, url, source_id, url_type) VALUES ('%s', '%s', '%s', '%s')" % (urn, self.url, self.source_id, self.urn_type))
            print('---')
            print('urn: %s' % urn)
            #print('r_component: %s' % r_component)
            print('url: %s' % self.url)
            print('urn_type: %s' % self.urn_type)
            print('<<<')
            self._update_history(urn=urn, r_component=None,
                                 url_old=None, url_new=self.url,
                                 url_type_old=None, url_type_new=self.urn_type)
            print('>>>')
            print(self.cursor, self.source_id, self.title, self.urn, self.url)

class SwedishUrnResolverFormatHandler(BaseFormatHandler):

    # This format gives all data in one go, there is never a resumption token.
    resumption_token = None

    def __init__(self, cursor, source_id, start_url, title, url_pattern, urn_type):
        BaseFormatHandler.__init__(self, cursor, source_id, start_url, title, url_pattern, urn_type)
        self.inside_identifier = False
        self.inside_url = False
        self.urns = set()
        self.urn_chars = ''
        self.url_chars = ''

    def startElement(self, name, attrs):
        if name == 'record':
            self.urn = None
            self.url = None
        elif name == 'identifier':
            self.inside_identifier = True
            self.urn_chars = ''
        elif name == 'url':
            self.inside_url = True
            self.url_chars = ''

    def endElement(self, name):
        if name == 'record':
            assert (self.urn is not None and self.url is not None)
            if self.urn in self.urns:
                log_error('%s: The source has same URN (%s) multiple times!' %
                          (self.title, self.urn))
            else:
                self.urns.add(self.urn)
                self.write_url()
        elif name == 'identifier':
            self.set_urn(self.urn_chars)
            self.inside_identifier = False
        elif name == 'url':
            self.set_url(self.url_chars)
            self.inside_url = False

    def characters(self, content):
        if self.inside_identifier:
            self.urn_chars += content
        elif self.inside_url:
            self.url_chars += content


class OAIPMHHandler(BaseFormatHandler):

    def __init__(self, cursor, source_id, start_url, title, url_pattern, urn_type):
        BaseFormatHandler.__init__(self, cursor, source_id, start_url, title, url_pattern, urn_type)
        self.inside_identifier = False
        self.inside_resumption_token = False
        self.resumption_token = None
        self.urns = set()

    def startElement(self, name, attrs):
        #print("startElement")
        if name == 'record':
            self.urn = None
            self.url = None
        elif name == 'dc:identifier':
            self.inside_identifier = True
            self.identifier = ''
        elif name == 'resumptionToken':
            self.inside_resumption_token = True
            self.resumption_chars = ''

    def endElement(self, name):
        if name == 'record':
            if (self.urn is not None and self.url is not None):
                if self.urn in self.urns:
                    log_error('%s: The source has same URN (%s) multiple times!' %
                              (self.title, self.urn))
                else:
                    self.urns.add(self.urn)
                    self.write_url()
        elif name == 'dc:identifier':
            self.dc_identifier()
        elif name == 'resumptionToken':
            self.resumption_token = self.resumption_chars
            self.inside_resumption_token = False

    def dc_identifier(self):
        if self.identifier.lower().startswith('urn'):
            self.set_urn(self.identifier)
        else:
            self.set_url(self.identifier)
        self.inside_identifier = False

    def characters(self, content):
        if self.inside_identifier:
            self.identifier += content
        elif self.inside_resumption_token:
            self.resumption_chars += content


# E-thesis community was originally in Doria, but it was later copied
# to Helda (but not removed from Doria (yet?)). Because we now harvest
# both Doria and Helda, we have to make sure that we get information
# about E-thesis items only from Helda. And that's the reason why there
# is this handler specifically for Doria.
# (And similarly Turun yliopisto has now their own instance.)
class DoriaHandler(OAIPMHHandler):
    def set_urn(self, urn):
        if urn and (urn not in ethesis_urns) and (urn not in turun_yliopisto_urns):
            self.urn = urn


# In Helda dc:identifier fields that should start with
# http://helda.helsinki.fi/handle/ start with
# http://hdl.handle.net/ so we have to fix that.
class HeldaHandler(OAIPMHHandler):
    def dc_identifier(self):
        self.inside_identifier = False
        if self.identifier.lower().startswith('urn'):
            self.set_urn(self.identifier)
        else:
            if (self.url and 
                self.url.startswith('http://helda.helsinki.fi/handle/')):
                return # We already have a nice URL, let's not ruin it!
            url = self.identifier.replace('http://hdl.handle.net/',
                                          'http://helda.helsinki.fi/handle/')
            self.set_url(url)


class OuluHandler(BaseFormatHandler):
    
    def __init__(self, cursor, source_id, start_url, title, url_pattern, urn_type):
        BaseFormatHandler.__init__(self, cursor, source_id, start_url, title, url_pattern, urn_type)
        self.inside_metadata = False
        self.inside_identifier = False
        self.inside_url = False
        self.inside_resumption_token = False
        self.resumption_token = None
        self.urns = set()
        self.urn_chars = ''
        self.url_chars = ''
        
    def startElement(self, name, attrs):
        if name == 'metadata':
            self.inside_metadata = True
            self.urn = None
            self.url = None
        elif name == 'identifier' and self.inside_metadata == True:
            self.inside_identifier = True
            self.urn_chars = ''
        elif name == 'url' and self.inside_metadata == True:
            self.inside_url = True
            self.url_chars = ''
        elif name == 'resumptionToken':
            self.inside_resumption_token = True
            self.resumption_chars = ''

    def endElement(self, name):
        if name == 'metadata':
            self.inside_metadata = False
            if (self.urn is not None and self.url is not None):
                if self.urn in self.urns:
                    log_error('%s: The source has same URN (%s) multiple times!' %
                              (self.title, self.urn))
                else:
                    self.urns.add(self.urn)
                    self.write_url()
        elif name == 'identifier':
            self.set_urn(self.urn_chars)
            self.inside_identifier = False
        elif name == 'url':
            self.set_url(self.url_chars)
            self.inside_url = False
        elif name == 'resumptionToken':
            self.resumption_token = self.resumption_chars
            self.inside_resumption_token = False

    def characters(self, content):
        if self.inside_identifier:
            self.urn_chars += content
        elif self.inside_url:
            self.url_chars += content
        elif self.inside_resumption_token:
            self.resumption_chars += content



def parse(cursor, source_id, title, url_pattern, format, start_url, resume_url=None,
          delay_between_requests=5, urn_type='normal'):

    handle = urlopen(start_url)
    parser = {'Helda': HeldaHandler,
              'Doria': DoriaHandler,
              'Oulu': OuluHandler,
              'OAI-PMH': OAIPMHHandler,
              'Swedish': SwedishUrnResolverFormatHandler}[format](cursor, source_id, start_url, title, url_pattern, urn_type)

    try:
        xml.sax.parse(handle, parser)
    except Exception:
        # There is one particular source which have had in few occasions an
        # invalid extra character (160 == 0xa0 == non-breaking space). So
        # let's try if we can fix the situation by removing that character:
        handle.close()
        s = urlopen(start_url).read()
        if '\xa0' not in s:
            raise # No such a luck. :-(
        else:
            # Ok, we have the chance to parse it again. Of course, this
            # may still fail.
            handle = StringIO(s.replace('\xa0', ''))
            xml.sax.parse(handle, parser)
    finally:
        handle.close()

    while (parser.resumption_token):
        assert resume_url != None
        log_debug('%s: Found a resumption token: %s' % 
                  (title, parser.resumption_token))
        
        time.sleep(delay_between_requests)
        handle = urlopen(resume_url + parser.resumption_token)
        parser.resumption_token = None
        xml.sax.parse(handle, parser)
        handle.close()
        
def harvest(source, db):
    start_time = datetime.now()

    log_info(u'Harvesting %s begins.' % source.title)

    try:
        success = False
        if source.format == 'Swedish':
            FIXparse(source.title, source.url_pattern,
                  source.format, source.start_url)
            success = True
        elif source.format in ('OAI-PMH', 'Doria', 'Helda'):
            start_url = source.start_url
            if source.is_next_run_full == False:
                start_url += '&from=%s' % \
                    source.last_successful_run_start.strftime('%Y-%m-%d')
            FIXparse(source.title, source.url_pattern,
                  source.format, start_url, source.resume_url,
                  source.delay_between_requests)
            success = True
        elif source.format == 'Oulu':
            FIXparse(source.title, source.url_pattern,
                  source.format, source.start_url, source.resume_url,
                  source.delay_between_requests)
            success = True
        else:
            log_error(u'Source %s has unknown format.' % source.title)

    except Exception as err:
        log_critical(u'Error harvesting source %s. (%s)' % (source.title, err))


    log_info(u'Harvesting %s ended.' % source.title)

def harvester_logger(q):
    logging.config.fileConfig('harvester_logging.conf')
    logger = logging.getLogger('harvester')
    while True:
        lvl, msg = q.get()
        logger.log(lvl, msg)

def init_logging():
    logger_queue = multiprocessing.Queue()
    def log_msg(lvl, q): return lambda msg: q.put((lvl, msg))
    global log_critical, log_error, log_info, log_debug
    log_critical = log_msg(logging.CRITICAL, logger_queue)
    log_error    = log_msg(logging.ERROR   , logger_queue)
    log_info     = log_msg(logging.INFO    , logger_queue)
    log_debug    = log_msg(logging.DEBUG   , logger_queue)

    logger_process = multiprocessing.Process(target=harvester_logger,
                                             args=(logger_queue,))
    logger_process.daemon = True
    logger_process.start()

def main(argv):
    if len(argv) != 2:
        err_print('Usage: %s [URN_source_title]' % argv[0])
        sys.exit(2)

    init_logging()

    try:
        connection = psycopg2.connect(user='uuno', database='uuno')
        cursor = connection.cursor()

        source_title = argv[1]
        for c in source_title:
            if c not in string.ascii_letters + string.digits + '_-':
                raise ValueError('Invalid source_title: %s' % source_title)
        
        cursor.execute("SELECT source_id, start_url, resume_url, format, source_type, url_pattern FROM source WHERE title = '%s'" % source_title)
        source_id, start_url, resume_url, source_format, urn_type, url_pattern = cursor.fetchone()

        print(source_id)
        print(start_url)
        print(source_format)
        
        parse(cursor, source_id, source_title, url_pattern, source_format,
              start_url, resume_url, 1, urn_type)

        exit_status = 0
    except (Exception, psycopg2.Error) as error:
        print('Error: %s' % error)
        exit_status = 1
    finally:
        if connection:
            cursor.close()
            connection.commit()
        
    sys.exit(exit_status)

if __name__ == "__main__":
    main(sys.argv)
