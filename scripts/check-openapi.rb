require 'yaml'

file = ARGV.fetch(0)
document = YAML.load_file(file)
abort 'Missing OpenAPI version' unless document['openapi']&.start_with?('3.')
abort 'Missing paths' unless document['paths'].is_a?(Hash)

names = []
references = []
visit = lambda do |node|
  case node
  when Hash
    references << node['$ref'] if node.key?('$ref')
    node.each_value { |child| visit.call(child) }
  when Array
    node.each { |child| visit.call(child) }
  end
end
visit.call(document)

document['paths'].each do |path, item|
  item.each do |method, operation|
    next unless %w[get post put patch delete].include?(method)
    name = operation['operationId']
    abort "Missing operationId: #{method.upcase} #{path}" if name.nil? || name.empty?
    abort "Duplicate operationId: #{name}" if names.include?(name)
    names << name
    abort "Missing responses: #{name}" unless operation['responses'].is_a?(Hash)
  end
end

references.each do |reference|
  abort "External references are not supported: #{reference}" unless reference.start_with?('#/')
  target = reference.delete_prefix('#/').split('/').reduce(document) { |parent, key| parent.is_a?(Hash) ? parent[key] : nil }
  abort "Unresolved reference: #{reference}" if target.nil?
end
puts "OpenAPI valid: #{names.size} operations, #{references.size} local references"
